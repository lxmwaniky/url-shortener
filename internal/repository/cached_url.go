package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lxmwaniky/url-shortener/internal/models"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type CachedURLRepository struct {
	postgresRepo URLRepository
	rdb          *redis.Client
	defaultTTL   time.Duration
	sfGroup      singleflight.Group
}

func NewCachedURLRepository(postgresRepo URLRepository, rdb *redis.Client, defaultTTL time.Duration) *CachedURLRepository {
	return &CachedURLRepository{
		postgresRepo: postgresRepo,
		rdb:          rdb,
		defaultTTL:   defaultTTL,
	}
}

func (r *CachedURLRepository) Create(ctx context.Context, originalURL string, customAlias string, expiresAt *time.Time) (*models.URL, error) {
	urlInfo, err := r.postgresRepo.Create(ctx, originalURL, customAlias, expiresAt)
	if err != nil {
		return nil, err
	}

	ttl := r.getTTL(urlInfo)
	if ttl > 0 {
		data, err := json.Marshal(urlInfo)
		if err == nil {
			key := fmt.Sprintf("url:%s", urlInfo.ShortCode)
			_ = r.rdb.Set(ctx, key, data, ttl).Err()
		}
	}

	return urlInfo, nil
}

func (r *CachedURLRepository) GetByShortCode(ctx context.Context, code string) (*models.URL, error) {
	key := fmt.Sprintf("url:%s", code)
	val, err := r.rdb.Get(ctx, key).Result()
	if err == nil {
		var urlInfo models.URL
		if err := json.Unmarshal([]byte(val), &urlInfo); err == nil {
			return &urlInfo, nil
		}
	}

	dbVal, err, _ := r.sfGroup.Do(code, func() (interface{}, error) {
		return r.postgresRepo.GetByShortCode(ctx, code)
	})
	if err != nil {
		return nil, err
	}

	urlInfo := dbVal.(*models.URL)

	ttl := r.getTTL(urlInfo)
	if ttl > 0 {
		data, err := json.Marshal(urlInfo)
		if err == nil {
			_ = r.rdb.Set(ctx, key, data, ttl).Err()
		}
	}

	return urlInfo, nil
}

func (r *CachedURLRepository) getTTL(urlInfo *models.URL) time.Duration {
	if urlInfo.ExpiresAt == nil {
		return r.defaultTTL
	}
	ttl := time.Until(*urlInfo.ExpiresAt)
	if ttl < 0 {
		return 0
	}
	return ttl
}
