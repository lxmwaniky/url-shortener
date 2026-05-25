package web

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lxmwaniky/url-shortener/internal/config"
	"github.com/lxmwaniky/url-shortener/internal/db"
)

func TestRedisRateLimiter(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Skipping Redis rate limiter test: %v", err)
	}

	rdb, err := db.ConnectRedis(cfg)
	if err != nil {
		t.Skipf("Skipping Redis rate limiter test: %v", err)
	}
	defer rdb.Close()

	t.Run("Test Throttling Behavior", func(t *testing.T) {
		ip := fmt.Sprintf("192.168.1.%d", time.Now().UnixNano()%255)
		limit := 3
		limiter := NewRedisRateLimiter(rdb, limit, 10*time.Second, "testwrite")

		for i := 1; i <= limit; i++ {
			if !limiter.Allow(ip) {
				t.Fatalf("expected request %d to be allowed", i)
			}
		}

		if limiter.Allow(ip) {
			t.Fatal("expected request after limit threshold to be throttled")
		}

		ctx := context.Background()
		now := time.Now()
		minuteKey := now.Unix() / 10
		key := fmt.Sprintf("rate_limit:testwrite:%s:%d", ip, minuteKey)
		_ = rdb.Del(ctx, key).Err()
	})
}
