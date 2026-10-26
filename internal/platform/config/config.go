package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env      string
	HTTPAddr string
	LogLevel slog.Level

	DBDSN             string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration

	RedisAddr     string
	RedisPassword string
	RedisDB       int
	CacheTTL      time.Duration

	JWTSecret  string
	JWTTTL     time.Duration
	BcryptCost int

	ShutdownTimeout     time.Duration
	RateLimitGlobalRate string
	RateLimitAuthRate   string
}

// Load читает конфигурацию из окружения. Секреты и DSN обязательны:
// приложение не должно запускаться с зашитыми в код значениями.
func Load() (*Config, error) {
	cfg := &Config{
		Env:               getEnv("APP_ENV", "local"),
		HTTPAddr:          getEnv("HTTP_ADDR", ":8080"),
		DBDSN:             os.Getenv("DB_DSN"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		DBMaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 25),
		RedisDB:           getInt("REDIS_DB", 0),
		ShutdownTimeout:   getDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		DBConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		CacheTTL:          getDuration("CACHE_TTL", 5*time.Minute),
		JWTTTL:            getDuration("JWT_TTL", 24*time.Hour),
		BcryptCost:        getInt("BCRYPT_COST", 12),
		RateLimitGlobalRate: getEnv("RATE_LIMIT_GLOBAL", "100-M"),
		RateLimitAuthRate:   getEnv("RATE_LIMIT_AUTH", "10-M"),
	}

	if strings.TrimSpace(cfg.DBDSN) == "" {
		return nil, fmt.Errorf("DB_DSN is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if cfg.DBMaxIdleConns > cfg.DBMaxOpenConns {
		cfg.DBMaxIdleConns = cfg.DBMaxOpenConns
	}

	level, err := parseLevel(getEnv("LOG_LEVEL", "info"))
	if err != nil {
		return nil, err
	}
	cfg.LogLevel = level

	return cfg, nil
}

func (c *Config) IsProduction() bool { return c.Env == "production" }

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func getDuration(key string, fallback time.Duration) time.Duration {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return v
}

func parseLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown LOG_LEVEL %q", raw)
	}
}
