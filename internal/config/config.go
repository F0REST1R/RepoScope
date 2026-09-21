package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment       string
	HTTPAddr          string
	DatabaseURL       string
	MigrationsDir     string
	GitHubBaseURL     string
	GitHubToken       string
	GitHubAPIVersion  string
	AllowedOrigins    []string
	HTTPClientTimeout time.Duration
	CacheTTL          time.Duration
	ShutdownTimeout   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Environment:       value("APP_ENV", "development"),
		HTTPAddr:          value("HTTP_ADDR", ":8080"),
		DatabaseURL:       value("DATABASE_URL", "postgres://reposcope:reposcope@localhost:5432/reposcope?sslmode=disable"),
		MigrationsDir:     value("MIGRATIONS_DIR", "migrations"),
		GitHubBaseURL:     strings.TrimRight(value("GITHUB_API_URL", "https://api.github.com"), "/"),
		GitHubToken:       os.Getenv("GITHUB_TOKEN"),
		GitHubAPIVersion:  value("GITHUB_API_VERSION", "2026-03-10"),
		AllowedOrigins:    split(value("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000")),
		HTTPClientTimeout: 12 * time.Second,
		CacheTTL:          2 * time.Minute,
		ShutdownTimeout:   10 * time.Second,
	}
	var err error
	if cfg.HTTPClientTimeout, err = duration("HTTP_CLIENT_TIMEOUT", cfg.HTTPClientTimeout); err != nil {
		return Config{}, err
	}
	if cfg.CacheTTL, err = duration("CACHE_TTL", cfg.CacheTTL); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = duration("SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func value(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func duration(key string, fallback time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}

func split(v string) []string {
	items := strings.Split(v, ",")
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// Int reads an optional integer environment variable and is kept small enough
// to be useful for future deployment tuning without bringing a config library.
func Int(key string, fallback int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return n, nil
}
