package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName   string
	AppEnv    string
	AppPort   string
	BaseURL   string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBTimezone string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	APIKey string

	RateLimitMax           int
	RateLimitWindowSeconds int
	RateLimitRedirectMax   int
	RateLimitRedirectWin   int

	ShortCodeLength int
	CacheTTLSeconds int
}

func Load() (*Config, error) {
	// .env opsional: di production semua env sudah diset oleh platform
	_ = godotenv.Load()

	cfg := &Config{
		AppName:   getEnv("APP_NAME", "Kliktaut"),
		AppEnv:    getEnv("APP_ENV", "development"),
		AppPort:   getEnv("APP_PORT", "3000"),
		BaseURL:   getEnv("BASE_URL", "http://localhost:3000"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5433"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     getEnv("DB_NAME", "kliktaut"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		DBTimezone: getEnv("DB_TIMEZONE", "Asia/Jakarta"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6381"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       0,

		APIKey: os.Getenv("API_KEY"),
	}

	var err error
	if cfg.RedisDB, err = getIntEnv("REDIS_DB", 0); err != nil {
		return nil, err
	}
	if cfg.RateLimitMax, err = getIntEnv("RATE_LIMIT_MAX", 60); err != nil {
		return nil, err
	}
	if cfg.RateLimitWindowSeconds, err = getIntEnv("RATE_LIMIT_WINDOW_SECONDS", 60); err != nil {
		return nil, err
	}
	if cfg.RateLimitRedirectMax, err = getIntEnv("RATE_LIMIT_REDIRECT_MAX", 300); err != nil {
		return nil, err
	}
	if cfg.RateLimitRedirectWin, err = getIntEnv("RATE_LIMIT_REDIRECT_WINDOW_SECONDS", 60); err != nil {
		return nil, err
	}
	if cfg.ShortCodeLength, err = getIntEnv("SHORT_CODE_LENGTH", 7); err != nil {
		return nil, err
	}
	if cfg.CacheTTLSeconds, err = getIntEnv("CACHE_TTL_SECONDS", 3600); err != nil {
		return nil, err
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API_KEY is required")
	}
	if cfg.CacheTTLSeconds <= 0 {
		return nil, fmt.Errorf("CACHE_TTL_SECONDS must be > 0")
	}
	if cfg.ShortCodeLength <= 0 {
		return nil, fmt.Errorf("SHORT_CODE_LENGTH must be > 0")
	}

	return cfg, nil
}

func (c *Config) DBDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode, c.DBTimezone,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number, got %q", key, v)
	}
	return n, nil
}
