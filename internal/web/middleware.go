package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			b := make([]byte, 16)
			if _, err := rand.Read(b); err != nil {
				requestID = fmt.Sprintf("%d", time.Now().UnixNano())
			} else {
				requestID = hex.EncodeToString(b)
			}
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "requestID", requestID)
		r = r.WithContext(ctx)

		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r)
	})
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		requestID := r.Context().Value("requestID")
		if requestID == nil {
			requestID = "unknown"
		}

		slog.Info("http request",
			"request_id", requestID,
			"ip", ip,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := r.Context().Value("requestID")
				if requestID == nil {
					requestID = "unknown"
				}
				slog.Error("panic recovered",
					"request_id", requestID,
					"error", err,
					"stack", string(debug.Stack()),
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func RateLimit(limiter *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if !limiter.Allow(ip) {
				slog.Warn("rate limit exceeded", "ip", ip, "path", r.URL.Path)
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
