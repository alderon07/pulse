package worker

import (
	"context"
	"log/slog"
	"time"

	"pulse/backend/internal/alerts"
	"pulse/backend/internal/repo"
)

type Runner struct {
	store          *repo.Store
	sender         *alerts.MultiSender
	logger         *slog.Logger
	overdueBatch   int
	deliveryBatch  int
	repeatInterval time.Duration
}

func NewRunner(store *repo.Store, sender *alerts.MultiSender, logger *slog.Logger, overdueBatch, deliveryBatch int, repeatInterval time.Duration) *Runner {
	if overdueBatch <= 0 {
		overdueBatch = 200
	}
	if deliveryBatch <= 0 {
		deliveryBatch = 100
	}
	return &Runner{
		store:          store,
		sender:         sender,
		logger:         logger,
		overdueBatch:   overdueBatch,
		deliveryBatch:  deliveryBatch,
		repeatInterval: repeatInterval,
	}
}

func (r *Runner) Tick(ctx context.Context) {
	updated, err := r.store.ProcessOverdueBatch(ctx, r.overdueBatch, r.repeatInterval)
	if err != nil {
		r.logger.Error("process overdue batch", "error", err)
	} else if updated > 0 {
		r.logger.Info("processed overdue batch", "updated_checks", updated)
	}

	deliveries, err := r.store.ClaimDeliveries(ctx, r.deliveryBatch)
	if err != nil {
		r.logger.Error("claim deliveries", "error", err)
		return
	}
	for _, delivery := range deliveries {
		deliveryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := r.sender.Send(deliveryCtx, delivery)
		cancel()

		if err == nil {
			if markErr := r.store.MarkDeliverySent(ctx, delivery.ID); markErr != nil {
				r.logger.Error("mark delivery sent", "delivery_id", delivery.ID, "error", markErr)
			}
			continue
		}

		if alerts.IsNonRetryable(err) {
			if markErr := r.store.MarkDeliveryFailed(ctx, delivery.ID, 99, err.Error()); markErr != nil {
				r.logger.Error("mark non-retryable failure", "delivery_id", delivery.ID, "error", markErr)
			}
			r.logger.Warn("non-retryable delivery failure", "delivery_id", delivery.ID, "error", err)
			continue
		}

		if markErr := r.store.MarkDeliveryFailed(ctx, delivery.ID, delivery.AttemptCount+1, err.Error()); markErr != nil {
			r.logger.Error("mark delivery failed", "delivery_id", delivery.ID, "error", markErr)
		}
		r.logger.Warn("retryable delivery failure", "delivery_id", delivery.ID, "error", err)
	}

	if deleted, err := r.store.CleanupIdempotencyKeys(ctx); err != nil {
		r.logger.Warn("cleanup idempotency keys failed", "error", err)
	} else if deleted > 0 {
		r.logger.Info("cleaned expired idempotency keys", "deleted", deleted)
	}
}
