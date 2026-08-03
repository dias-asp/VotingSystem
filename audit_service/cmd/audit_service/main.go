package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"audit_service/internal/config"
	"audit_service/internal/database"
	"audit_service/internal/health"
	"audit_service/internal/httpmw"
	"audit_service/internal/repository/postgres"
	"audit_service/internal/service"
	"audit_service/internal/transport/httpapi"
	"audit_service/internal/transport/kafka"
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

	blockRepo := postgres.NewBlockRepository(db)
	recordRepo := postgres.NewRecordRepository(db)

	blockBuilder := service.NewBlockBuilder(recordRepo, blockRepo, cfg.BlockSize)
	auditor := service.NewAuditor(recordRepo)
	prover := service.NewProver(blockRepo, recordRepo)
	chain := service.NewChainVerifier(db)

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopicAnonymized, cfg.KafkaGroupID, auditor)
	defer func() { _ = consumer.Close() }()

	healthz := health.Handler(2*time.Second, map[string]health.Checker{
		"db":    health.DB(db),
		"kafka": health.Kafka(cfg.KafkaBrokers),
	})
	router := httpapi.NewRouter(httpapi.NewHandler(prover, chain), healthz)
	wrapped := httpmw.Chain(router, httpmw.RequestID, httpmw.Logger(logger))
	server := httpapi.NewServer(cfg.HTTPAddr, wrapped)

	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("http server starting", "addr", cfg.HTTPAddr)
		errCh <- server.Start()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("kafka consumer starting", "topic", cfg.KafkaTopicAnonymized, "group", cfg.KafkaGroupID)
		errCh <- consumer.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("block builder starting", "interval", cfg.BlockInterval, "size", cfg.BlockSize)
		errCh <- blockBuilder.Run(ctx, cfg.BlockInterval)
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
