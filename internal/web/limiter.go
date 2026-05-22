package web

import (
	"sync"
	"time"
)

type TokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

type IPRateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*TokenBucket
	refillRate float64
	maxTokens  float64
}

func NewIPRateLimiter(limit int, period time.Duration) *IPRateLimiter {
	rate := float64(limit) / period.Seconds()
	return &IPRateLimiter{
		buckets:    make(map[string]*TokenBucket),
		refillRate: rate,
		maxTokens:  float64(limit),
	}
}

func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	bucket, exists := l.buckets[ip]
	if !exists {
		l.buckets[ip] = &TokenBucket{
			tokens:     l.maxTokens,
			maxTokens:  l.maxTokens,
			refillRate: l.refillRate,
			lastRefill: now,
		}
		return true
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * bucket.refillRate
	if bucket.tokens > bucket.maxTokens {
		bucket.tokens = bucket.maxTokens
	}
	bucket.lastRefill = now

	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}
