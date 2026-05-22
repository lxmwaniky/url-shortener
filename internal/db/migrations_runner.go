package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"

	"gitlab.com/lxmwaniky/url-shortener/internal/db/migrations"
)

type MigrationRunner struct {
	db *sql.DB
}

func NewMigrationRunner(db *sql.DB) *MigrationRunner {
	return &MigrationRunner{db: db}
}

func (r *MigrationRunner) MigrateUp() error {
	ctx := context.Background()

	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	files, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to read embedded migrations: %w", err)
	}

	type migrationFile struct {
		version int
		name    string
	}

	var upMigrations []migrationFile
	versionRegex := regexp.MustCompile(`^(\d+)_.+\.up\.sql$`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		matches := versionRegex.FindStringSubmatch(file.Name())
		if len(matches) == 2 {
			version, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			upMigrations = append(upMigrations, migrationFile{
				version: version,
				name:    file.Name(),
			})
		}
	}

	sort.Slice(upMigrations, func(i, j int) bool {
		return upMigrations[i].version < upMigrations[j].version
	})

	for _, m := range upMigrations {
		var exists bool
		err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", m.version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration version %d: %w", m.version, err)
		}

		if exists {
			continue
		}

		content, err := fs.ReadFile(migrations.FS, m.name)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", m.name, err)
		}

		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", m.name, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", m.version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction for migration %d: %w", m.version, err)
		}
	}

	return nil
}
