package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           int
	DBDSN          string
	AdminToken     string
	CachePrewarm   int
	CreateRateRPS  float64
	CreateRateBurst int
	BaseURL        string // used for returning absolute short URLs
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getint(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getfloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return def
}

func Load() Config {
	return Config{
		Port:            getint("PORT", 8080),
		DBDSN:           getenv("DB_DSN", "file:shorty.db?_foreign_keys=on"),
		AdminToken:      getenv("ADMIN_TOKEN", ""),
		CachePrewarm:    getint("CACHE_PREWARM", 100),
		CreateRateRPS:   getfloat("CREATE_RATE_RPS", 2.0),
		CreateRateBurst: getint("CREATE_RATE_BURST", 5),
		BaseURL:         getenv("BASE_URL", ""),
	}
}
