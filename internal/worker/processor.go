package worker

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
)

const (
	QueueName           = "vitalcache:jobs"
	ProcessingQueueName = "vitalcache:jobs:processing"
)

type Job struct {
	Type    string `json:"type"`    // e.g., "generate_pdf", "send_email"
	Payload string `json:"payload"` // JSON stringified data
}

type Processor struct {
	rdb *redis.Client
	log *zap.Logger
	wg  *sync.WaitGroup
}

func NewProcessor(rdb *redis.Client) *Processor {
	return &Processor{
		rdb: rdb,
		log: logger.Named("worker"),
		wg:  &sync.WaitGroup{},
	}
}

func (p *Processor) Wait() {
	p.wg.Wait()
}

func (p *Processor) Start(ctx context.Context) {
	p.log.Info("Starting Redis background worker...", zap.String("queue", QueueName))

	go func() {
		for {
			select {
			case <-ctx.Done():
				p.log.Info("Shutting down worker...")
				return
			default:
				// BRPopLPush blocks and reliably moves the job to processing queue
				res, err := p.rdb.BRPopLPush(ctx, QueueName, ProcessingQueueName, 2*time.Second).Result()
				if err != nil {
					if err != redis.Nil && err != context.Canceled {
						p.log.Error("Redis pop error", zap.Error(err))
					}
					continue
				}

				if res != "" {
					var job Job
					if err := json.Unmarshal([]byte(res), &job); err != nil {
						p.log.Error("Failed to parse job", zap.Error(err))
						p.rdb.LRem(context.Background(), ProcessingQueueName, 1, res)
						continue
					}

					p.wg.Add(1)
					go func(jobData string, parsedJob Job) {
						defer p.wg.Done()
						p.processJob(parsedJob)
						p.rdb.LRem(context.Background(), ProcessingQueueName, 1, jobData)
					}(res, job)
				}
			}
		}
	}()
}

func (p *Processor) processJob(job Job) {
	start := time.Now()
	p.log.Info("Processing job", zap.String("type", job.Type))

	switch job.Type {
	case "generate_pdf":
		// Mock PDF Generation
		time.Sleep(1 * time.Second)
		p.log.Info("Prescription PDF generated successfully", zap.String("payload", job.Payload))

	case "send_email":
		// Mock Email Sending
		time.Sleep(500 * time.Millisecond)
		p.log.Info("Email sent successfully", zap.String("payload", job.Payload))

	default:
		p.log.Warn("Unknown job type", zap.String("type", job.Type))
	}

	p.log.Info("Job completed", zap.String("type", job.Type), zap.Duration("duration", time.Since(start)))
}
