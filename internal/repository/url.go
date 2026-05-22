package repository

import (
	"context"
	"time"

	"gitlab.com/lxmwaniky/url-shortener/internal/models"
)

type URLRepository interface {
	Create(ctx context.Context, originalURL string, customAlias string, expiresAt *time.Time) (*models.URL, error)
	GetByShortCode(ctx context.Context, code string) (*models.URL, error)
}
