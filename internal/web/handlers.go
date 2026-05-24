package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gitlab.com/lxmwaniky/url-shortener/internal/repository"
)

type ShortenRequest struct {
	OriginalURL string `json:"original_url"`
	CustomAlias string `json:"custom_alias,omitempty"`
}

type ShortenResponse struct {
	ShortURL  string    `json:"short_url"`
	ShortCode string    `json:"short_code"`
	ExpiresAt time.Time `json:"expires_at"`
}

type DBConnection interface {
	PingContext(ctx context.Context) error
}

type Handlers struct {
	repo    repository.URLRepository
	db      DBConnection
	baseURI string
}

func NewHandlers(repo repository.URLRepository, db DBConnection, baseURI string) *Handlers {
	return &Handlers{
		repo:    repo,
		db:      db,
		baseURI: baseURI,
	}
}

// isPrivateIP checks if a host string represents a private/internal IP address
// Returns true for localhost, loopback, private ranges (RFC1918), and link-local addresses
func isPrivateIP(host string) bool {
	// Remove port if present
	if colon := strings.LastIndex(host, ":"); colon != -1 {
		host = host[:colon]
	}

	// Check for localhost
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Not a valid IP, could be a domain name - we'll allow it (could be enhanced with DNS resolution)
		return false
	}

	// Check for private IP ranges
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for private networks (RFC1918): 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	if ipInNet(ip, "10.0.0.0/8") || ipInNet(ip, "172.16.0.0/12") || ipInNet(ip, "192.168.0.0/16") {
		return true
	}

	// Check for carrier-grade NAT (RFC6598): 100.64.0.0/10
	if ipInNet(ip, "100.64.0.0/10") {
		return true
	}

	return false
}

// ipInNet checks if an IP address is within a CIDR network
func ipInNet(ip net.IP, cidr string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return network.Contains(ip)
}

func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	endpoints := map[string]interface{}{
		"service": "url-shortener",
		"endpoints": map[string]string{
			"GET /":        "Show available service endpoints",
			"GET /health":  "Database connectivity and SRE health check",
			"POST /shorten": "Shorten a long URL. Accepts JSON body with 'original_url' and optional 'custom_alias'",
			"GET /{code}":  "Redirect to the original URL associated with the short code",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoints)
}

func (h *Handlers) Shorten(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.OriginalURL == "" {
		http.Error(w, "original_url is required", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.ParseRequestURI(req.OriginalURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		http.Error(w, "invalid original_url format", http.StatusBadRequest)
		return
	}

	if isPrivateIP(parsedURL.Host) {
		http.Error(w, "shortening private or internal URLs is forbidden", http.StatusBadRequest)
		return
	}

	expiresAt := time.Now().Add(30 * 24 * time.Hour) // Cap lifespan to exactly 1 month

	created, err := h.repo.Create(r.Context(), req.OriginalURL, req.CustomAlias, &expiresAt)
	if err != nil {
		if errors.Is(err, repository.ErrAliasAlreadyExists) {
			http.Error(w, "custom alias already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to shorten URL", http.StatusInternalServerError)
		return
	}

	resp := ShortenResponse{
		ShortURL:  fmt.Sprintf("%s/%s", h.baseURI, created.ShortCode),
		ShortCode: created.ShortCode,
		ExpiresAt: expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) Redirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		http.Error(w, "missing short code", http.StatusBadRequest)
		return
	}

	urlInfo, err := h.repo.GetByShortCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, repository.ErrURLNotFound) {
			http.Error(w, "short URL not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to retrieve short URL", "code", code, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if urlInfo.ExpiresAt != nil && time.Now().After(*urlInfo.ExpiresAt) {
		http.Error(w, "short URL has expired", http.StatusGone)
		return
	}

	http.Redirect(w, r, urlInfo.OriginalURL, http.StatusFound)
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
