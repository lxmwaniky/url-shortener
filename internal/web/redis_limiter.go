package web

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	rdb         *redis.Client
	limit       int64
	period      time.Duration
	limiterType string
}

func NewRedisRateLimiter(rdb *redis.Client, limit int, period time.Duration, limiterType string) *RedisRateLimiter {
	return &RedisRateLimiter{
		rdb:         rdb,
		limit:       int64(limit),
		period:      period,
		limiterType: limiterType,
	}
}

func (l *RedisRateLimiter) Allow(ip string) bool {
	ctx := context.Background()
	now := time.Now()
	minuteKey := now.Unix() / int64(l.period.Seconds())
	key := fmt.Sprintf("rate_limit:%s:%s:%d", l.limiterType, ip, minuteKey)

	pipe := l.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, l.period+5*time.Second)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return true
	}

	return incr.Val() <= l.limit
}
