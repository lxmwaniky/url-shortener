package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lxmwaniky/url-shortener/internal/config"
)

func Connect(cfg *config.Config) (*sql.DB, error) {
	var dsn string

	if len(cfg.DBHost) > 0 && cfg.DBHost[0] == '/' {
		dsn = fmt.Sprintf("postgres://%s:%s@localhost/%s?sslmode=%s&host=%s",
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBName,
			cfg.DBSSLMode,
			cfg.DBHost,
		)
	} else {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBName,
			cfg.DBSSLMode,
		)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
