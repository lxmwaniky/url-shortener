package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/lxmwaniky/url-shortener/internal/models"
)

type mockRedisPingable struct {
	pingFunc func(ctx context.Context) *redis.StatusCmd
}

func (m *mockRedisPingable) Ping(ctx context.Context) *redis.StatusCmd {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return redis.NewStatusResult("PONG", nil)
}

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

type mockDB struct {
	pingFunc func(context.Context) error
}

func (m *mockDB) PingContext(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

func TestHealthHandler(t *testing.T) {
	mockRepo := &MockURLRepository{}

	t.Run("Both Healthy", func(t *testing.T) {
		handlers := NewHandlers(
			mockRepo,
			&mockDB{pingFunc: func(context.Context) error { return nil }},
			&mockRedisPingable{pingFunc: func(ctx context.Context) *redis.StatusCmd {
				return redis.NewStatusResult("PONG", nil)
			}},
			"http://localhost:8080",
		)

		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		handlers.Health(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", resp["status"])
		}

		components := resp["components"].(map[string]interface{})
		if components["database"] != "up" || components["redis"] != "up" {
			t.Errorf("Expected both components up, got %v", resp["components"])
		}
	})

	t.Run("Database Unhealthy", func(t *testing.T) {
		handlers := NewHandlers(
			mockRepo,
			&mockDB{pingFunc: func(context.Context) error { return errors.New("db error") }},
			&mockRedisPingable{pingFunc: func(ctx context.Context) *redis.StatusCmd {
				return redis.NewStatusResult("PONG", nil)
			}},
			"http://localhost:8080",
		)

		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		handlers.Health(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, rr.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp["status"] != "unhealthy" {
			t.Errorf("Expected status 'unhealthy', got %v", resp["status"])
		}

		components := resp["components"].(map[string]interface{})
		if components["database"] != "down" || components["redis"] != "up" {
			t.Errorf("Expected database down and redis up, got %v", resp["components"])
		}
	})

	t.Run("Redis Unhealthy", func(t *testing.T) {
		handlers := NewHandlers(
			mockRepo,
			&mockDB{pingFunc: func(context.Context) error { return nil }},
			&mockRedisPingable{pingFunc: func(ctx context.Context) *redis.StatusCmd {
				return redis.NewStatusResult("", errors.New("redis connection timeout"))
			}},
			"http://localhost:8080",
		)

		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		handlers.Health(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, rr.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp["status"] != "unhealthy" {
			t.Errorf("Expected status 'unhealthy', got %v", resp["status"])
		}

		components := resp["components"].(map[string]interface{})
		if components["database"] != "up" || components["redis"] != "down" {
			t.Errorf("Expected database up and redis down, got %v", resp["components"])
		}
	})
}

func TestIndexHandler(t *testing.T) {
	mockRepo := &MockURLRepository{}
	handlers := NewHandlers(mockRepo, nil, nil, "http://localhost:8080")

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handlers.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["service"] != "url-shortener" {
		t.Errorf("Expected service 'url-shortener', got %v", resp["service"])
	}
}

func TestShortenHandlerValidation(t *testing.T) {
	mockRepo := &MockURLRepository{
		CreateFunc: func(ctx context.Context, originalURL string, customAlias string, expiresAt *time.Time) (*models.URL, error) {
			return &models.URL{
				ID:          1,
				ShortCode:   "abc",
				OriginalURL: originalURL,
				CreatedAt:   time.Now(),
				ExpiresAt:   expiresAt,
			}, nil
		},
	}

	handlers := NewHandlers(mockRepo, nil, nil, "http://localhost:8080")

	body := `{"original_url": ""}`
	req := httptest.NewRequest("POST", "/shorten", strings.NewReader(body))
	rr := httptest.NewRecorder()

	handlers.Shorten(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d for missing URL, got %d", http.StatusBadRequest, rr.Code)
	}

	body = `{"original_url": "not-a-valid-url"}`
	req = httptest.NewRequest("POST", "/shorten", strings.NewReader(body))
	rr = httptest.NewRecorder()

	handlers.Shorten(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d for invalid URL, got %d", http.StatusBadRequest, rr.Code)
	}

	body = `{"original_url": "http://localhost:5432"}`
	req = httptest.NewRequest("POST", "/shorten", strings.NewReader(body))
	rr = httptest.NewRecorder()

	handlers.Shorten(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d for private IP, got %d", http.StatusBadRequest, rr.Code)
	}

	body = `{"original_url": "https://google.com"}`
	req = httptest.NewRequest("POST", "/shorten", strings.NewReader(body))
	rr = httptest.NewRecorder()

	handlers.Shorten(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d for valid URL, got %d", http.StatusCreated, rr.Code)
	}

	var resp ShortenResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ShortCode != "abc" {
		t.Errorf("Expected short code 'abc', got %q", resp.ShortCode)
	}
	if resp.ShortURL != "http://localhost:8080/abc" {
		t.Errorf("Expected short URL 'http://localhost:8080/abc', got %q", resp.ShortURL)
	}
}
