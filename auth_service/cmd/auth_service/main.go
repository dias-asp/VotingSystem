package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"auth_service/internal/config"
	"auth_service/internal/database"
	"auth_service/internal/health"
	"auth_service/internal/httpmw"
	"auth_service/internal/passwords"
	"auth_service/internal/repository/postgres"
	"auth_service/internal/service"
	"auth_service/internal/tokens"
	"auth_service/internal/transport/httpapi"
)

const bcryptCost = 12

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := sql.Open("pgx", cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, pingCancel := context.WithTimeout(ctx, 10*time.Second)
	if err := db.PingContext(pingCtx); err != nil {
		pingCancel()
		return err
	}
	pingCancel()

	if err := database.Migrate(ctx, db, cfg.MigrationsDir); err != nil {
		return err
	}

	privateKey, err := tokens.LoadOrGeneratePrivateKey(cfg.JWTPrivateKeyPath)
	if err != nil {
		return err
	}

	users := postgres.NewUserRepository(db)
	refresh := postgres.NewRefreshTokenRepository(db)
	issuer := tokens.NewIssuer(privateKey, cfg.JWTIssuer, cfg.JWTKeyID)
	hasher := passwords.NewBcryptHasher(bcryptCost)
	auth := service.NewAuth(users, refresh, issuer, hasher, cfg.AccessTTL, cfg.RefreshTTL, cfg.AdminEmail)

	healthz := health.Handler(2*time.Second, map[string]health.Checker{
		"db": health.DB(db),
	})
	router := httpapi.NewRouter(httpapi.NewHandler(auth, issuer), healthz)
	wrapped := httpmw.Chain(router, httpmw.RequestID, httpmw.Logger(logger))
	server := httpapi.NewServer(cfg.HTTPAddr, wrapped)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server starting", "addr", cfg.HTTPAddr)
		errCh <- server.Start()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
