package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr             string
	PostgresDSN          string
	MigrationsDir        string
	KafkaBrokers         []string
	KafkaTopicVotes      string
	KafkaTopicAnonymized string
	KafkaGroupID         string
	ShutdownTimeout      time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:             getEnv("HTTP_ADDR", ":8081"),
		PostgresDSN:          getEnv("POSTGRES_DSN", ""),
		MigrationsDir:        getEnv("MIGRATIONS_DIR", "./migrations"),
		KafkaBrokers:         splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopicVotes:      getEnv("KAFKA_TOPIC_VOTES", "vote.events"),
		KafkaTopicAnonymized: getEnv("KAFKA_TOPIC_ANONYMIZED", "vote.anonymized"),
		KafkaGroupID:         getEnv("KAFKA_GROUP_ID", "anonym_service"),
		ShutdownTimeout:      getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
	}
	if cfg.PostgresDSN == "" {
		return cfg, fmt.Errorf("POSTGRES_DSN is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
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
