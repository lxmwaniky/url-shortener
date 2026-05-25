package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lxmwaniky/url-shortener/internal/config"
	"github.com/lxmwaniky/url-shortener/internal/db"
	"github.com/lxmwaniky/url-shortener/internal/models"
)

type MockURLRepository struct {
	CreateFunc         func(ctx context.Context, originalURL string, customAlias string, expiresAt *time.Time) (*models.URL, error)
	GetByShortCodeFunc func(ctx context.Context, code string) (*models.URL, error)
}

func (m *MockURLRepository) Create(ctx context.Context, originalURL string, customAlias string, expiresAt *time.Time) (*models.URL, error) {
	return m.CreateFunc(ctx, originalURL, customAlias, expiresAt)
}

func (m *MockURLRepository) GetByShortCode(ctx context.Context, code string) (*models.URL, error) {
	return m.GetByShortCodeFunc(ctx, code)
}

func TestCachedURLRepository(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Skipping cached repository integration test: %v", err)
	}

	rdb, err := db.ConnectRedis(cfg)
	if err != nil {
		t.Skipf("Skipping cached repository integration test: %v", err)
	}
	defer rdb.Close()

	ctx := context.Background()

	t.Run("Test Cache Miss then Hit on GetByShortCode", func(t *testing.T) {
		code := fmt.Sprintf("testcode-%d", time.Now().UnixNano())
		key := fmt.Sprintf("url:%s", code)
		_ = rdb.Del(ctx, key).Err()

		dbCalls := 0
		expectedURL := &models.URL{
			ID:          999,
			ShortCode:   code,
			OriginalURL: "https://example.com/target",
			CreatedAt:   time.Now().Truncate(time.Second),
		}

		mockRepo := &MockURLRepository{
			GetByShortCodeFunc: func(ctx context.Context, c string) (*models.URL, error) {
				dbCalls++
				if c != code {
					return nil, errors.New("unexpected code requested")
				}
				return expectedURL, nil
			},
		}

		cachedRepo := NewCachedURLRepository(mockRepo, rdb, 10*time.Second)

		firstGet, err := cachedRepo.GetByShortCode(ctx, code)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dbCalls != 1 {
			t.Errorf("expected 1 call to database repo, got %d", dbCalls)
		}
		if firstGet.ID != expectedURL.ID || firstGet.OriginalURL != expectedURL.OriginalURL {
			t.Errorf("mismatched URL returned on first call")
		}

		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			t.Fatalf("expected key to be set in redis cache: %v", err)
		}
		var cachedURL models.URL
		if err := json.Unmarshal([]byte(val), &cachedURL); err != nil {
			t.Fatalf("failed to unmarshal cached JSON: %v", err)
		}
		if cachedURL.ID != expectedURL.ID {
			t.Errorf("mismatched cached URL id")
		}

		secondGet, err := cachedRepo.GetByShortCode(ctx, code)
		if err != nil {
			t.Fatalf("unexpected error on second get: %v", err)
		}
		if dbCalls != 1 {
			t.Errorf("expected database calls to remain 1 (cache hit), got %d", dbCalls)
		}
		if secondGet.ID != expectedURL.ID {
			t.Errorf("mismatched cached URL returned")
		}

		_ = rdb.Del(ctx, key).Err()
	})

	t.Run("Test Pre-warming Cache on Create", func(t *testing.T) {
		code := fmt.Sprintf("createcode-%d", time.Now().UnixNano())
		key := fmt.Sprintf("url:%s", code)
		_ = rdb.Del(ctx, key).Err()

		expectedURL := &models.URL{
			ID:          888,
			ShortCode:   code,
			OriginalURL: "https://example.com/prewarm",
			CreatedAt:   time.Now().Truncate(time.Second),
		}

		mockRepo := &MockURLRepository{
			CreateFunc: func(ctx context.Context, u string, alias string, exp *time.Time) (*models.URL, error) {
				return expectedURL, nil
			},
		}

		cachedRepo := NewCachedURLRepository(mockRepo, rdb, 10*time.Second)

		_, err := cachedRepo.Create(ctx, "https://example.com/prewarm", "", nil)
		if err != nil {
			t.Fatalf("unexpected create error: %v", err)
		}

		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			t.Fatalf("expected key to be set in redis cache immediately on create: %v", err)
		}
		var cachedURL models.URL
		if err := json.Unmarshal([]byte(val), &cachedURL); err != nil {
			t.Fatalf("failed to unmarshal cached JSON: %v", err)
		}
		if cachedURL.ID != expectedURL.ID {
			t.Errorf("mismatched pre-warmed URL id")
		}

		_ = rdb.Del(ctx, key).Err()
	})
}
