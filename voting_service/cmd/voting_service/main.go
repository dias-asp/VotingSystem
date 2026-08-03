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

	"sync"

	"voting_service/internal/auth"
	"voting_service/internal/config"
	"voting_service/internal/database"
	"voting_service/internal/health"
	"voting_service/internal/httpmw"
	"voting_service/internal/outbox"
	"voting_service/internal/repository/postgres"
	"voting_service/internal/service"
	"voting_service/internal/transport/httpapi"
	"voting_service/internal/transport/kafka"
)

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

	keys := auth.NewKeySet(cfg.JWKSURL)
	if err := waitForJWKS(ctx, keys); err != nil {
		return err
	}
	go keys.RunRefresher(ctx, cfg.JWKSRefreshEvery)
	verifier := auth.NewVerifier(keys, cfg.JWTIssuer)

	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopicVotes)
	defer func() { _ = producer.Close() }()

	repo := postgres.NewVoteRepository(db)
	pollsRepo := postgres.NewPollRepository(db)
	polls := service.NewPolls(pollsRepo)
	voting := service.NewVoting(repo, pollsRepo)

	outboxStore := postgres.NewOutboxRepository(db)
	dispatcher := outbox.NewDispatcher(outboxStore, producer, cfg.OutboxBatchSize, cfg.OutboxInterval)

	healthz := health.Handler(2*time.Second, map[string]health.Checker{
		"db":    health.DB(db),
		"kafka": health.Kafka(cfg.KafkaBrokers),
	})
	router := httpapi.NewRouter(httpapi.NewHandler(voting, polls), auth.Middleware(verifier), healthz)
	wrapped := httpmw.Chain(router, httpmw.RequestID, httpmw.Logger(logger))
	server := httpapi.NewServer(cfg.HTTPAddr, wrapped)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("http server starting", "addr", cfg.HTTPAddr)
		errCh <- server.Start()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("outbox dispatcher starting", "interval", cfg.OutboxInterval, "batch", cfg.OutboxBatchSize)
		errCh <- dispatcher.Run(ctx)
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		stop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		wg.Wait()
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	wg.Wait()
	return nil
}

func waitForJWKS(ctx context.Context, keys *auth.KeySet) error {
	const maxAttempts = 12
	backoff := time.Second
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := keys.Refresh(fetchCtx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		slog.Warn("jwks fetch failed; retrying", "attempt", i+1, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 10*time.Second {
			backoff *= 2
		}
	}
	return lastErr
}
