package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// Config holds application configuration loaded from the environment.
type Config struct {
	DatabaseURL string
	JWTSecret   string
	JWTExpiry   time.Duration
	ServerPort  string
	BcryptCost  int
	GinMode     string
}

// Load reads configuration from environment variables (and optional .env file).
// DATABASE_URL and JWT_SECRET are required.
func Load() (*Config, error) {
	_ = godotenv.Load()
	// Optional: when running the API on the host, copy `.env.host.example` to `.env.host`
	// with DATABASE_URL pointing at localhost (Compose must publish postgres:5432).
	if _, err := os.Stat(".env.host"); err == nil {
		_ = godotenv.Overload(".env.host")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	jwtExpiry := 24 * time.Hour
	if v := os.Getenv("JWT_EXPIRY"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("JWT_EXPIRY: %w", err)
		}
		jwtExpiry = d
	}

	serverPort := getenvDefault("SERVER_PORT", "8080")

	const minBcryptCost = 12
	bcryptCost := minBcryptCost
	if v := os.Getenv("BCRYPT_COST"); v != "" {
		c, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("BCRYPT_COST: %w", err)
		}
		if c < bcrypt.MinCost || c > bcrypt.MaxCost {
			return nil, fmt.Errorf("BCRYPT_COST must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
		}
		if c < minBcryptCost {
			return nil, fmt.Errorf("BCRYPT_COST must be at least %d (TaskFlow requirement)", minBcryptCost)
		}
		bcryptCost = c
	}

	ginMode := getenvDefault("GIN_MODE", "release")
	if ginMode != "debug" && ginMode != "release" && ginMode != "test" {
		return nil, fmt.Errorf("GIN_MODE must be one of debug, release, test")
	}

	return &Config{
		DatabaseURL: databaseURL,
		JWTSecret:   jwtSecret,
		JWTExpiry:   jwtExpiry,
		ServerPort:  serverPort,
		BcryptCost:  bcryptCost,
		GinMode:     ginMode,
	}, nil
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
