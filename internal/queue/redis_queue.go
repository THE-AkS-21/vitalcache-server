package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client *redis.Client
	queue  string
}

func NewRedisQueue(client *redis.Client) Client {
	return &RedisQueue{
		client: client,
		queue:  "vitalcache:queue:prescriptions",
	}
}

func (q *RedisQueue) EnqueuePrescription(ctx context.Context, patientID uint, filePath, fileName string) error {
	p := prescriptionPayload{PatientID: patientID, FilePath: filePath, FileName: fileName}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return q.client.RPush(ctx, q.queue, b).Err()
}

func (q *RedisQueue) Dequeue(ctx context.Context) (string, error) {
	// Non-blocking pop or short timeout
	res, err := q.client.BLPop(ctx, 1*time.Second, q.queue).Result()
	if err != nil {
		return "", err
	}
	// res[0] is key, res[1] is value
	return res[1], nil
}

func (q *RedisQueue) StartWorker(ctx context.Context) error {
	go func() {
		slog.Info("redis queue worker started")
		for {
			select {
			case <-ctx.Done():
				slog.Info("redis queue worker stopped")
				return
			default:
				// BLPop blocks until an item is available or timeout
				res, err := q.client.BLPop(ctx, 5*time.Second, q.queue).Result()
				if err != nil {
					if err != redis.Nil {
						// slog.Warn("redis queue pop failed", "err", err)
					}
					continue
				}
				// res[0] is key, res[1] is value
				var p prescriptionPayload
				if err := json.Unmarshal([]byte(res[1]), &p); err != nil {
					slog.Warn("queue payload decode failed", "err", err)
					continue
				}

				// Simulate processing
				time.Sleep(300 * time.Millisecond)

				if err := os.Remove(p.FilePath); err != nil && !os.IsNotExist(err) {
					slog.Warn("temp file cleanup failed", "path", p.FilePath, "err", err)
				} else {
					slog.Info("processed prescription (redis)", "patient_id", p.PatientID, "file", p.FileName)
				}
			}
		}
	}()
	return nil
}
