package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"gitlab.com/lxmwaniky/url-shortener/internal/config"
	"gitlab.com/lxmwaniky/url-shortener/internal/db"
	"gitlab.com/lxmwaniky/url-shortener/internal/repository"
	"gitlab.com/lxmwaniky/url-shortener/internal/service"
	"gitlab.com/lxmwaniky/url-shortener/internal/web"
)

func main() {
	// Initialize SRE Structured Logging (JSON)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("starting url-shortener api server")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	database, err := db.Connect(cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	slog.Info("applying database migrations")
	migrationsRunner := db.NewMigrationRunner(database)
	if err := migrationsRunner.MigrateUp(); err != nil {
		slog.Error("failed to apply database migrations", "error", err)
		os.Exit(1)
	}

	// Automated SRE Cleaner: background task to purge expired links once a day
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		ctx := context.Background()
		for range ticker.C {
			slog.Info("starting automated background purge of expired urls")
			result, err := database.ExecContext(ctx, "DELETE FROM urls WHERE expires_at < NOW()")
			if err != nil {
				slog.Error("failed to purge expired urls", "error", err)
				continue
			}
			rows, _ := result.RowsAffected()
			slog.Info("completed automated background purge", "deleted_rows", rows)
		}
	}()

	feistel := service.NewFeistel(cfg.FeistelSeed)
	encoder := service.NewBase62Encoder()
	repo := repository.NewPostgresURLRepository(database, feistel, encoder)

	baseURI := fmt.Sprintf("http://localhost:%s", cfg.Port)
	if cfg.Env == "production" {
		baseURI = "https://url-shortener.com" // Update when live domain is ready
	}

	handlers := web.NewHandlers(repo, database, baseURI)

	// Rate Limiting: 10 writes (link creations) per client IP per minute
	limiter := web.NewIPRateLimiter(10, 1*time.Minute)

	// Setup go-chi router and register middlewares
	r := chi.NewRouter()
	r.Use(web.Recovery)
	r.Use(web.Logger)

	r.Get("/", handlers.Index)
	r.Get("/health", handlers.Health)
	r.Get("/{code}", handlers.Redirect)

	// Rate limit is mounted only on the write path (POST /shorten) to protect storage
	r.With(web.RateLimit(limiter)).Post("/shorten", handlers.Shorten)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Listen for SIGINT / SIGTERM signals for Graceful SRE Shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("server is listening", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server listener crashed", "error", err)
			os.Exit(1)
		}
	}()

	sig := <-shutdownChan
	slog.Info("received termination signal, initiating graceful shutdown", "signal", sig.String())

	// Give active requests 15 seconds to safely drain/complete before forcing exit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to crash during graceful shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server shut down gracefully, exiting safely")
}
