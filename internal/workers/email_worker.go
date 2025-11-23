package workers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	"github.com/THE-AkS-21/vitalcache-server/internal/queue"
)

type EmailWorker struct {
	q queue.Client
}

func NewEmailWorker(q queue.Client) *EmailWorker {
	return &EmailWorker{q: q}
}

func (w *EmailWorker) Start(ctx context.Context) {
	slog.Info("starting email worker")
	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("stopping email worker")
				return
			default:
				// Blocking pop from queue
				task, err := w.q.Dequeue(ctx)
				if err != nil {
					// If queue is empty or error, sleep briefly
					time.Sleep(1 * time.Second)
					continue
				}

				w.processTask(ctx, task)
			}
		}
	}()
}

func (w *EmailWorker) processTask(ctx context.Context, task string) {
	// In a real implementation, we would parse the task (e.g., JSON)
	// and send an email via SES/SendGrid/SMTP.
	// For now, we just log it as per the requirement.

	var p domain.Prescription
	if err := json.Unmarshal([]byte(task), &p); err == nil {
		slog.Info("processing email task",
			"type", "prescription_created",
			"patient_id", p.PatientID,
			"file_url", p.FileURL,
			"status", "sent (stubbed)",
		)
	} else {
		slog.Info("processing email task", "payload", task, "status", "sent (stubbed)")
	}
}
