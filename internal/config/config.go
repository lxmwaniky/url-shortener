package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var (
	SkipEnvLoad = false
)

type Config struct {
	Port            string
	Env             string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBSSLMode       string
	FeistelSeed     uint32
	CleanupInterval string
	BaseURL         string
	RedisHost       string
	RedisPort       string
	RedisPassword   string
	RedisDB         int
}

func Load() (*Config, error) {
	loadEnv()

	port := getEnv("PORT", "8080")
	if err := validatePort(port); err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	env := getEnv("ENV", "local")

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		return nil, errors.New("DB_HOST environment variable is required")
	}

	dbPort := getEnv("DB_PORT", "5432")
	if err := validatePort(dbPort); err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

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

	cleanupInterval := getEnv("CLEANUP_INTERVAL", "24h")
	if _, err := time.ParseDuration(cleanupInterval); err != nil {
		return nil, fmt.Errorf("invalid CLEANUP_INTERVAL: must be a valid duration string: %w", err)
	}

	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	if err := validatePort(redisPort); err != nil {
		return nil, fmt.Errorf("invalid REDIS_PORT: %w", err)
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisDBStr := getEnv("REDIS_DB", "0")
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	baseURL := getEnv("BASE_URL", "http://localhost:"+port)

	return &Config{
		Port:            port,
		Env:             env,
		DBHost:          dbHost,
		DBPort:          dbPort,
		DBUser:          dbUser,
		DBPassword:      dbPassword,
		DBName:          dbName,
		DBSSLMode:       dbSSLMode,
		FeistelSeed:     uint32(seed),
		CleanupInterval: cleanupInterval,
		BaseURL:         baseURL,
		RedisHost:       redisHost,
		RedisPort:       redisPort,
		RedisPassword:   redisPassword,
		RedisDB:         redisDB,
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

func validatePort(port string) error {
	if port == "" {
		return errors.New("port cannot be empty")
	}

	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("port must be a number: %w", err)
	}

	portNum, _ := strconv.Atoi(port)
	if portNum < 1 || portNum > 65535 {
		return errors.New("port must be between 1 and 65535")
	}

	return nil
}
