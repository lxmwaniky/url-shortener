//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/lxmwaniky/url-shortener/internal/config"
)

func TestDatabaseConnectionAndMigrations(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Skipping database integration test: %v", err)
	}

	db, err := Connect(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	t.Log("Database connection verified successfully!")

	runner := NewMigrationRunner(db)
	if err := runner.MigrateUp(); err != nil {
		t.Fatalf("Failed to run database migrations: %v", err)
	}
	t.Log("Database migrations applied successfully!")

	ctx := context.Background()
	var exists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'urls'
		)
	`).Scan(&exists)

	if err != nil {
		t.Fatalf("Failed to query information_schema: %v", err)
	}
	if !exists {
		t.Error("Expected 'urls' table to exist, but it was not found")
	}

	t.Log("Verified that the 'urls' table exists in the database schema!")
}
