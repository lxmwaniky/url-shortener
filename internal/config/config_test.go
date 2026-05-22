package config

import (
	"os"
	"testing"
)

func TestConfigLoadSuccess(t *testing.T) {
	defer clearEnv()()

	os.Setenv("PORT", "9090")
	os.Setenv("ENV", "production")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "app_user")
	os.Setenv("DB_PASSWORD", "secret123")
	os.Setenv("DB_NAME", "shortener_db")
	os.Setenv("DB_SSLMODE", "require")
	os.Setenv("FEISTEL_SEED", "987654321")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected configuration load to succeed, got error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("Expected Port to be '9090', got %q", cfg.Port)
	}
	if cfg.Env != "production" {
		t.Errorf("Expected Env to be 'production', got %q", cfg.Env)
	}
	if cfg.DBHost != "db.example.com" {
		t.Errorf("Expected DBHost to be 'db.example.com', got %q", cfg.DBHost)
	}
	if cfg.DBUser != "app_user" {
		t.Errorf("Expected DBUser to be 'app_user', got %q", cfg.DBUser)
	}
	if cfg.DBName != "shortener_db" {
		t.Errorf("Expected DBName to be 'shortener_db', got %q", cfg.DBName)
	}
	if cfg.FeistelSeed != 987654321 {
		t.Errorf("Expected FeistelSeed to be 987654321, got %d", cfg.FeistelSeed)
	}
}

func TestConfigLoadMissingRequiredFields(t *testing.T) {
	defer clearEnv()()

	os.Setenv("PORT", "8080")

	_, err := Load()
	if err == nil {
		t.Error("Expected configuration load to fail due to missing required fields, but it succeeded")
	}
}

func TestConfigLoadInvalidSeed(t *testing.T) {
	defer clearEnv()()

	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "password")
	os.Setenv("DB_NAME", "db")
	os.Setenv("FEISTEL_SEED", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Error("Expected configuration load to fail due to invalid FEISTEL_SEED format, but it succeeded")
	}
}

func clearEnv() func() {
	keys := []string{
		"PORT", "ENV", "DB_HOST", "DB_PORT", "DB_USER",
		"DB_PASSWORD", "DB_NAME", "DB_SSLMODE", "FEISTEL_SEED",
	}

	backup := make(map[string]string)
	present := make(map[string]bool)

	for _, key := range keys {
		if val, ok := os.LookupEnv(key); ok {
			backup[key] = val
			present[key] = true
		}
		os.Unsetenv(key)
	}

	SkipEnvLoad = true

	return func() {
		SkipEnvLoad = false
		for _, key := range keys {
			if present[key] {
				os.Setenv(key, backup[key])
			} else {
				os.Unsetenv(key)
			}
		}
	}
}
