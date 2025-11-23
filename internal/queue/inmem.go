package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"
)

const TypePrescriptionSend = "prescription:send"

type Client interface {
	EnqueuePrescription(ctx context.Context, patientID uint, filePath, fileName string) error
	StartWorker(ctx context.Context) error
	Dequeue(ctx context.Context) (string, error)
}

type inmemQ struct {
	ch   chan []byte
	once sync.Once
}

type prescriptionPayload struct {
	PatientID uint   `json:"patient_id"`
	FilePath  string `json:"file_path"`
	FileName  string `json:"file_name"`
}

func NewInMemory() Client {
	return &inmemQ{ch: make(chan []byte, 1024)}
}

func (q *inmemQ) EnqueuePrescription(ctx context.Context, patientID uint, filePath, fileName string) error {
	p := prescriptionPayload{PatientID: patientID, FilePath: filePath, FileName: fileName}
	b, _ := json.Marshal(p)
	select {
	case q.ch <- b:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *inmemQ) Dequeue(ctx context.Context) (string, error) {
	select {
	case b := <-q.ch:
		return string(b), nil
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return "", os.ErrNotExist // Empty queue
	}
}

func (q *inmemQ) StartWorker(ctx context.Context) error {
	q.once.Do(func() {
		go func() {
			slog.Info("in-memory queue worker started")
			for {
				select {
				case <-ctx.Done():
					slog.Info("in-memory queue worker stopped")
					return
				case b := <-q.ch:
					var p prescriptionPayload
					if err := json.Unmarshal(b, &p); err != nil {
						slog.Warn("queue payload decode failed", "err", err)
						continue
					}
					// Simulate some processing
					time.Sleep(300 * time.Millisecond)

					// Just log and cleanup the temp file (no email)
					if err := os.Remove(p.FilePath); err != nil && !os.IsNotExist(err) {
						slog.Warn("temp file cleanup failed", "path", p.FilePath, "err", err)
					} else {
						slog.Info("processed prescription (no-email mode)", "patient_id", p.PatientID, "file", p.FileName)
					}
				}
			}
		}()
	})
	return nil
}
