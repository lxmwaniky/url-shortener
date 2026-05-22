package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/lxmwaniky/url-shortener/internal/config"
	"gitlab.com/lxmwaniky/url-shortener/internal/db"
	"gitlab.com/lxmwaniky/url-shortener/internal/service"
)

func TestPostgresURLRepository(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Skipping repository integration test: %v", err)
	}

	database, err := db.Connect(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Ensure migrations are run so table structure is present
	runner := db.NewMigrationRunner(database)
	if err := runner.MigrateUp(); err != nil {
		t.Fatalf("Failed to run database migrations: %v", err)
	}

	feistel := service.NewFeistel(cfg.FeistelSeed)
	encoder := service.NewBase62Encoder()
	repo := NewPostgresURLRepository(database, feistel, encoder)

	ctx := context.Background()

	_, err = database.ExecContext(ctx, "TRUNCATE TABLE urls RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("Failed to clean up database: %v", err)
	}

	t.Run("Test Create and Get Standard URL", func(t *testing.T) {
		original := "https://example.com/very/long/url/path"
		expiresAt := time.Now().Add(24 * time.Hour)

		created, err := repo.Create(ctx, original, "", &expiresAt)
		if err != nil {
			t.Fatalf("Failed to create short URL: %v", err)
		}

		if created.ShortCode == "" {
			t.Error("Expected generated short code to be non-empty")
		}
		if created.OriginalURL != original {
			t.Errorf("Expected OriginalURL to be %q, got %q", original, created.OriginalURL)
		}

		// Verify retrieval (Primary Key high-speed path)
		retrieved, err := repo.GetByShortCode(ctx, created.ShortCode)
		if err != nil {
			t.Fatalf("Failed to retrieve URL by short code: %v", err)
		}

		if retrieved.ID != created.ID {
			t.Errorf("Expected ID %d, got %d", created.ID, retrieved.ID)
		}
		if retrieved.OriginalURL != original {
			t.Errorf("Expected OriginalURL %q, got %q", original, retrieved.OriginalURL)
		}
	})

	t.Run("Test Create and Get Custom Alias", func(t *testing.T) {
		original := "https://another-example.com"
		alias := "custom-test-alias"

		created, err := repo.Create(ctx, original, alias, nil)
		if err != nil {
			t.Fatalf("Failed to create custom URL: %v", err)
		}

		if created.ShortCode != alias {
			t.Errorf("Expected short code to be %q, got %q", alias, created.ShortCode)
		}

		// Verify retrieval (Fallback string lookup path)
		retrieved, err := repo.GetByShortCode(ctx, alias)
		if err != nil {
			t.Fatalf("Failed to retrieve custom URL: %v", err)
		}

		if retrieved.OriginalURL != original {
			t.Errorf("Expected OriginalURL %q, got %q", original, retrieved.OriginalURL)
		}
	})

	t.Run("Test Unique Alias Violation", func(t *testing.T) {
		original := "https://url1.com"
		alias := "duplicate-alias"

		_, err := repo.Create(ctx, original, alias, nil)
		if err != nil {
			t.Fatalf("Failed to create initial custom URL: %v", err)
		}

		// Attempt duplicate insert
		_, err = repo.Create(ctx, "https://url2.com", alias, nil)
		if !errors.Is(err, ErrAliasAlreadyExists) {
			t.Errorf("Expected error ErrAliasAlreadyExists, got %v", err)
		}
	})

	t.Run("Test Non-Existent URL", func(t *testing.T) {
		_, err := repo.GetByShortCode(ctx, "non-existent-code")
		if !errors.Is(err, ErrURLNotFound) {
			t.Errorf("Expected error ErrURLNotFound, got %v", err)
		}
	})
}
