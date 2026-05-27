package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lxmwaniky/url-shortener/internal/models"
	"github.com/lxmwaniky/url-shortener/internal/service"
)

var (
	ErrAliasAlreadyExists = errors.New("short code alias already exists")
	ErrURLNotFound        = errors.New("short url not found")
)

type PostgresURLRepository struct {
	db      *sql.DB
	feistel *service.Feistel
	encoder *service.Base62Encoder
}

func NewPostgresURLRepository(db *sql.DB, feistel *service.Feistel, encoder *service.Base62Encoder) *PostgresURLRepository {
	return &PostgresURLRepository{
		db:      db,
		feistel: feistel,
		encoder: encoder,
	}
}

func (r *PostgresURLRepository) Create(ctx context.Context, originalURL string, customAlias string, expiresAt *time.Time) (*models.URL, error) {
	if customAlias != "" {
		var id uint64
		var createdAt time.Time

		query := `
			INSERT INTO urls (short_code, original_url, expires_at) 
			VALUES ($1, $2, $3) 
			RETURNING id, created_at
		`
		err := r.db.QueryRowContext(ctx, query, customAlias, originalURL, expiresAt).Scan(&id, &createdAt)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return nil, ErrAliasAlreadyExists
			}
			return nil, fmt.Errorf("failed to create custom url: %w", err)
		}

		return &models.URL{
			ID:          id,
			ShortCode:   customAlias,
			OriginalURL: originalURL,
			CreatedAt:   createdAt,
			ExpiresAt:   expiresAt,
		}, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var id uint64
	err = tx.QueryRowContext(ctx, "SELECT nextval('urls_id_seq')").Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate sequence ID: %w", err)
	}

	shuffled := r.feistel.Encrypt(id)
	shortCode := r.encoder.Encode(shuffled)

	var createdAt time.Time
	query := `
		INSERT INTO urls (id, short_code, original_url, expires_at) 
		VALUES ($1, $2, $3, $4) 
		RETURNING created_at
	`
	err = tx.QueryRowContext(ctx, query, id, shortCode, originalURL, expiresAt).Scan(&createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert short url: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.URL{
		ID:          id,
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		CreatedAt:   createdAt,
		ExpiresAt:   expiresAt,
	}, nil
}

func (r *PostgresURLRepository) GetByShortCode(ctx context.Context, code string) (*models.URL, error) {
	shuffled, err := r.encoder.Decode(code)
	if err == nil {
		id := r.feistel.Decrypt(shuffled)
		if id <= 9223372036854775807 {
			var url models.URL
			query := `
				SELECT id, short_code, original_url, created_at, expires_at 
				FROM urls 
				WHERE id = $1
			`
			err = r.db.QueryRowContext(ctx, query, id).Scan(&url.ID, &url.ShortCode, &url.OriginalURL, &url.CreatedAt, &url.ExpiresAt)
			if err == nil {
				return &url, nil
			}

			if !errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("failed to query url by primary key: %w", err)
			}
		}
	}

	var url models.URL
	query := `
		SELECT id, short_code, original_url, created_at, expires_at 
		FROM urls 
		WHERE short_code = $1
	`
	err = r.db.QueryRowContext(ctx, query, code).Scan(&url.ID, &url.ShortCode, &url.OriginalURL, &url.CreatedAt, &url.ExpiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrURLNotFound
		}
		return nil, fmt.Errorf("failed to query url by short code index: %w", err)
	}

	return &url, nil
}
