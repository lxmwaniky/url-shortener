package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	SkipEnvLoad = false
)

type Config struct {
	Port        string
	Env         string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	FeistelSeed uint32
}

func Load() (*Config, error) {
	// Automatically load .env file from the root or parent directories
	loadEnv()

	port := getEnv("PORT", "8080")
	env := getEnv("ENV", "local")

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		return nil, errors.New("DB_HOST environment variable is required")
	}

	dbPort := getEnv("DB_PORT", "5432")

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		return nil, errors.New("DB_USER environment variable is required")
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		return nil, errors.New("DB_PASSWORD environment variable is required")
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return nil, errors.New("DB_NAME environment variable is required")
	}

	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	seedStr := os.Getenv("FEISTEL_SEED")
	if seedStr == "" {
		return nil, errors.New("FEISTEL_SEED environment variable is required")
	}

	seed, err := strconv.ParseUint(seedStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid FEISTEL_SEED: must be a positive 32-bit integer: %w", err)
	}

	return &Config{
		Port:        port,
		Env:         env,
		DBHost:      dbHost,
		DBPort:      dbPort,
		DBUser:      dbUser,
		DBPassword:  dbPassword,
		DBName:      dbName,
		DBSSLMode:   dbSSLMode,
		FeistelSeed: uint32(seed),
	}, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func loadEnv() {
	if SkipEnvLoad {
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	// Search upwards for .env file
	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Load(envPath)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}
