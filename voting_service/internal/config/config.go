package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr         string
	PostgresDSN      string
	MigrationsDir    string
	KafkaBrokers     []string
	KafkaTopicVotes  string
	ShutdownTimeout  time.Duration
	JWKSURL          string
	JWTIssuer        string
	JWKSRefreshEvery time.Duration
	OutboxInterval   time.Duration
	OutboxBatchSize  int
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:         getEnv("HTTP_ADDR", ":8080"),
		PostgresDSN:      getEnv("POSTGRES_DSN", ""),
		MigrationsDir:    getEnv("MIGRATIONS_DIR", "./migrations"),
		KafkaBrokers:     splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopicVotes:  getEnv("KAFKA_TOPIC_VOTES", "vote.events"),
		ShutdownTimeout:  getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		JWKSURL:          getEnv("JWKS_URL", ""),
		JWTIssuer:        getEnv("JWT_ISSUER", "voting-system-auth"),
		JWKSRefreshEvery: getEnvDuration("JWKS_REFRESH_EVERY", 5*time.Minute),
		OutboxInterval:   getEnvDuration("OUTBOX_INTERVAL", time.Second),
		OutboxBatchSize:  getEnvInt("OUTBOX_BATCH_SIZE", 100),
	}
	if cfg.PostgresDSN == "" {
		return cfg, fmt.Errorf("POSTGRES_DSN is required")
	}
	if cfg.JWKSURL == "" {
		return cfg, fmt.Errorf("JWKS_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
