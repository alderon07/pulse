package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv                    string
	HTTPAddr                  string
	DatabaseURL               string
	WorkerTick                time.Duration
	OverdueBatchSize          int
	DeliveryBatchSize         int
	AlertRepeatInterval       time.Duration
	PingRatePerMinute         int
	PingBurst                 int
	InvalidTokenRatePerMinute int
	InvalidTokenBurst         int
	WebhookTimeout            time.Duration
	WebhookHMACSecret         string
	IdempotencyTTL            time.Duration
	JWTIssuer                 string
	JWTAudience               string
	JWTJWKSURL                string
	JWTHS256Secret            string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:                    getOrDefault("APP_ENV", "development"),
		HTTPAddr:                  getOrDefault("HTTP_ADDR", ":8080"),
		WorkerTick:                time.Duration(getIntOrDefault("WORKER_TICK_SECONDS", 60)) * time.Second,
		OverdueBatchSize:          getIntOrDefault("OVERDUE_BATCH_SIZE", 200),
		DeliveryBatchSize:         getIntOrDefault("DELIVERY_BATCH_SIZE", 100),
		AlertRepeatInterval:       time.Duration(getIntOrDefault("ALERT_REPEAT_MINUTES", 60)) * time.Minute,
		PingRatePerMinute:         getIntOrDefault("PING_RATE_PER_MIN", 60),
		PingBurst:                 getIntOrDefault("PING_RATE_BURST", 20),
		InvalidTokenRatePerMinute: getIntOrDefault("INVALID_TOKEN_RATE_PER_MIN", 30),
		InvalidTokenBurst:         getIntOrDefault("INVALID_TOKEN_BURST", 10),
		WebhookTimeout:            time.Duration(getIntOrDefault("WEBHOOK_TIMEOUT_SECONDS", 5)) * time.Second,
		WebhookHMACSecret:         os.Getenv("WEBHOOK_HMAC_SECRET"),
		IdempotencyTTL:            time.Duration(getIntOrDefault("IDEMPOTENCY_TTL_HOURS", 24)) * time.Hour,
		DatabaseURL:               os.Getenv("DATABASE_URL"),
		JWTIssuer:                 os.Getenv("JWT_ISSUER"),
		JWTAudience:               os.Getenv("JWT_AUDIENCE"),
		JWTJWKSURL:                os.Getenv("JWT_JWKS_URL"),
		JWTHS256Secret:            os.Getenv("JWT_HS256_SECRET"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.WorkerTick <= 0 {
		return Config{}, fmt.Errorf("WORKER_TICK_SECONDS must be > 0")
	}
	return cfg, nil
}

func getOrDefault(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func getIntOrDefault(k string, fallback int) int {
	if v := os.Getenv(k); v != "" {
		parsed, err := strconv.Atoi(v)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
