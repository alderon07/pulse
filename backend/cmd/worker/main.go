package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pulse/backend/internal/alerts"
	"pulse/backend/internal/config"
	"pulse/backend/internal/db"
	"pulse/backend/internal/repo"
	"pulse/backend/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	store := repo.New(database)
	sender := alerts.NewMultiSender(
		alerts.NewEmailSender(logger),
		alerts.NewWebhookSender(cfg.WebhookTimeout, cfg.WebhookHMACSecret, logger),
	)
	runner := worker.NewRunner(store, sender, logger, cfg.OverdueBatchSize, cfg.DeliveryBatchSize, cfg.AlertRepeatInterval)

	ticker := time.NewTicker(cfg.WorkerTick)
	defer ticker.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("worker started", "tick", cfg.WorkerTick.String())
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), cfg.WorkerTick)
			runner.Tick(ctx)
			cancel()
		case <-sigCh:
			logger.Info("worker stopping")
			return
		}
	}
}
