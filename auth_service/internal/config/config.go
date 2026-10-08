package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddr          string
	PostgresDSN       string
	MigrationsDir     string
	JWTPrivateKeyPath string
	JWTIssuer         string
	JWTKeyID          string
	AccessTTL         time.Duration
	RefreshTTL        time.Duration
	ShutdownTimeout   time.Duration
	AdminEmail        string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:          getEnv("HTTP_ADDR", ":8083"),
		PostgresDSN:       getEnv("POSTGRES_DSN", ""),
		MigrationsDir:     getEnv("MIGRATIONS_DIR", "./migrations"),
		JWTPrivateKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", "/etc/auth/private.pem"),
		JWTIssuer:         getEnv("JWT_ISSUER", "ngvs-auth"),
		JWTKeyID:          getEnv("JWT_KEY_ID", "auth-1"),
		AccessTTL:         getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:        getEnvDuration("JWT_REFRESH_TTL", 30*24*time.Hour),
		ShutdownTimeout:   getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		AdminEmail:        getEnv("ADMIN_EMAIL", ""),
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
