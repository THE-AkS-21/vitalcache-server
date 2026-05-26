package worker

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// StartBillingAggregator runs a daily job to compress old billings into weekly aggregates.
func StartBillingAggregator(ctx context.Context, pgPool *pgxpool.Pool, logger *zap.SugaredLogger) {
	logger.Info("Starting periodic billing aggregator worker")

	// Run once immediately (with slight delay so server boots first)
	go func() {
		time.Sleep(5 * time.Second)
		runAggregation(ctx, pgPool, logger)
	}()

	ticker := time.NewTicker(24 * time.Hour)

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("Shutting down billing aggregator")
				ticker.Stop()
				return
			case <-ticker.C:
				runAggregation(ctx, pgPool, logger)
			}
		}
	}()
}

func runAggregation(ctx context.Context, pgPool *pgxpool.Pool, logger *zap.SugaredLogger) {
	logger.Info("Running billing aggregation job")

	tx, err := pgPool.Begin(ctx)
	if err != nil {
		logger.Errorw("Failed to begin transaction for billing aggregation", "error", err)
		return
	}
	defer tx.Rollback(ctx)

	// Aggregate by doctor and week. We only aggregate PAID billings older than 7 days.
	query := `
		WITH weekly_aggs AS (
			SELECT 
				doctor_id, 
				date_trunc('week', created_at) AS period_start,
				date_trunc('week', created_at) + interval '6 days' AS period_end,
				SUM(amount) as total_amount,
				COUNT(DISTINCT patient_id) as total_patients,
				array_agg(id) as aggregated_billing_ids
			FROM billings
			WHERE status = 'PAID' AND created_at < NOW() - INTERVAL '7 days'
			GROUP BY doctor_id, date_trunc('week', created_at)
		),
		inserted AS (
			INSERT INTO billing_aggregates (period_type, period_start, period_end, doctor_id, total_amount, total_patients)
			SELECT 'weekly', period_start, period_end, doctor_id, total_amount, total_patients
			FROM weekly_aggs
			RETURNING id
		)
		DELETE FROM billings
		WHERE id IN (
			SELECT unnest(aggregated_billing_ids) FROM weekly_aggs
		);
	`
	tag, err := tx.Exec(ctx, query)
	if err != nil {
		logger.Errorw("Failed to aggregate weekly billings", "error", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Errorw("Failed to commit billing aggregation tx", "error", err)
		return
	}

	logger.Infow("Billing aggregation completed", "rows_affected_deleted", tag.RowsAffected())
}
