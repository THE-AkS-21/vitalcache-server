// Package queue provides a Redis Streams-based reliable async queue for VitalCache.
//
// Redis Streams (vs the old list-based queue) offer:
//   - Consumer groups: multiple workers can read from the same stream
//   - Acknowledgements: messages are only deleted after explicit ACK
//   - Crash recovery: pending unacked messages are re-delivered on restart
//   - Built-in persistence: survives Redis restarts (AOF/RDB)
//
// Well-known stream names — all workers use these constants:
//
//	StreamEmail            → outbound email tasks
//	StreamNotifications    → in-app / push notifications
//	StreamPrescriptions    → async prescription post-processing
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
)

// Stream name constants — keep producers and consumers in sync.
const (
	StreamEmail         = "vitalcache:email"
	StreamNotifications = "vitalcache:notifications"
	StreamPrescriptions = "vitalcache:prescriptions"
	StreamDLQ           = "vitalcache:dlq" // Dead Letter Queue for poison pills

	workerBlockDuration = 5 * time.Second // XREADGROUP block timeout
	claimIdleThreshold  = 1 * time.Minute // XAUTOCLAIM: reclaim messages idle this long
	maxRetryCount       = 3               // delivery limit before routing to DLQ
)

// Message is a decoded stream entry.
type Message struct {
	ID     string            `json:"id"`
	Stream string            `json:"stream"`
	Fields map[string]string `json:"fields"`
}

// HandlerFunc processes a single stream message. Return nil to ACK; return an
// error to NACK (the message will be re-tried up to maxRetryCount times).
type HandlerFunc func(ctx context.Context, msg Message) error

// RedisStreamQueue wraps a Redis client for stream-based pub/sub.
type RedisStreamQueue struct {
	rdb *redis.Client
	log *zap.Logger
}

// NewRedisStreamQueue returns a new queue backed by the given Redis client.
func NewRedisStreamQueue(rdb *redis.Client) *RedisStreamQueue {
	return &RedisStreamQueue{rdb: rdb, log: logger.Named("queue.redis")}
}

// Publish encodes payload as JSON and appends it to the named stream.
// The Redis key is the stream name (e.g. "vitalcache:email").
//
//	err := q.Publish(ctx, queue.StreamEmail, map[string]any{
//	    "to":      "user@example.com",
//	    "subject": "Your prescription is ready",
//	    "body":    "...",
//	})
func (q *RedisStreamQueue) Publish(ctx context.Context, stream string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("queue.Publish marshal: %w", err)
	}
	if err := q.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: 10_000, // cap stream length to prevent unbounded growth
		Approx: true,
		Values: map[string]any{"data": string(data)},
	}).Err(); err != nil {
		return fmt.Errorf("queue.Publish xadd: %w", err)
	}
	q.log.Debug("published message", zap.String("stream", stream))
	return nil
}

// EnsureConsumerGroup creates the consumer group if it does not already exist.
// Call this at worker startup before Consume.
func (q *RedisStreamQueue) EnsureConsumerGroup(ctx context.Context, stream, group string) error {
	err := q.rdb.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("queue.EnsureConsumerGroup: %w", err)
	}
	return nil
}

// Consume starts a blocking consumer loop for the given stream and group.
// It processes messages by calling handler and ACKs them on success.
// On handler error, messages are NACKed and re-delivered up to maxRetryCount.
// The loop runs until ctx is cancelled.
//
//	go q.Consume(ctx, queue.StreamEmail, "email-group", "worker-1", emailHandler)
func (q *RedisStreamQueue) Consume(ctx context.Context, stream, group, consumer string, handler HandlerFunc) error {
	if err := q.EnsureConsumerGroup(ctx, stream, group); err != nil {
		return err
	}
	q.log.Info("consumer started",
		zap.String("stream", stream),
		zap.String("group", group),
		zap.String("consumer", consumer),
	)

	for {
		select {
		case <-ctx.Done():
			q.log.Info("consumer stopping", zap.String("stream", stream), zap.String("group", group))
			return ctx.Err()
		default:
		}

		// ── XAUTOCLAIM sweep ─────────────────────────────────────────────────
		// Re-claim messages that have been sitting idle in another consumer's PEL
		// for longer than claimIdleThreshold. This recovers from crashed workers.
		claimed, _, err := q.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   stream,
			Group:    group,
			Consumer: consumer,
			MinIdle:  claimIdleThreshold,
			Start:    "0-0",
			Count:    10,
		}).Result()
		if err != nil && err != redis.Nil {
			q.log.Warn("XAUTOCLAIM error", zap.Error(err))
		}
		for _, msg := range claimed {
			q.process(ctx, stream, group, msg, handler)
		}

		// ── Read new messages ────────────────────────────────────────────────
		entries, err := q.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{stream, ">"},
			Count:    10,
			Block:    workerBlockDuration,
			NoAck:    false,
		}).Result()

		if err == redis.Nil || err == context.DeadlineExceeded {
			continue
		}
		if err != nil {
			q.log.Error("XREADGROUP error", zap.Error(err))
			time.Sleep(2 * time.Second) // backoff on transient errors
			continue
		}

		for _, entry := range entries {
			for _, msg := range entry.Messages {
				q.process(ctx, stream, group, msg, handler)
			}
		}
	}
}

// process handles a single stream message and ACKs/NACKs accordingly.
// On handler error it checks the delivery count via XPENDING. If the message
// has been delivered >= maxRetryCount times it is a poison pill: it gets
// written to the DLQ stream and ACKed so the consumer can make forward progress.
func (q *RedisStreamQueue) process(ctx context.Context, stream, group string, raw redis.XMessage, handler HandlerFunc) {
	fields := make(map[string]string, len(raw.Values))
	for k, v := range raw.Values {
		if s, ok := v.(string); ok {
			fields[k] = s
		}
	}
	msg := Message{ID: raw.ID, Stream: stream, Fields: fields}

	if err := handler(ctx, msg); err != nil {
		q.log.Warn("message handler error",
			zap.String("id", msg.ID),
			zap.String("stream", stream),
			zap.Error(err),
		)

		// Check delivery count to detect poison pills.
		pending, pendingErr := q.rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: stream,
			Group:  group,
			Start:  raw.ID,
			End:    raw.ID,
			Count:  1,
		}).Result()

		if pendingErr == nil && len(pending) > 0 && pending[0].RetryCount >= int64(maxRetryCount) {
			q.log.Error("poison pill detected — routing to DLQ",
				zap.String("id", raw.ID),
				zap.String("stream", stream),
				zap.Int64("delivery_count", pending[0].RetryCount),
			)
			// Write to DLQ with original metadata preserved.
			dlqValues := map[string]any{
				"origin_stream":  stream,
				"origin_id":      raw.ID,
				"origin_group":   group,
				"delivery_count": pending[0].RetryCount,
				"last_error":     err.Error(),
			}
			for k, v := range fields {
				dlqValues["payload_"+k] = v
			}
			if addErr := q.rdb.XAdd(ctx, &redis.XAddArgs{
				Stream: StreamDLQ,
				MaxLen: 5_000,
				Approx: true,
				Values: dlqValues,
			}).Err(); addErr != nil {
				q.log.Error("failed to write to DLQ", zap.Error(addErr))
				// Do NOT ACK — leave in PEL to try again rather than silently losing it.
				return
			}
			// ACK the original so the group can make forward progress.
			if ackErr := q.rdb.XAck(ctx, stream, group, raw.ID).Err(); ackErr != nil {
				q.log.Warn("XACK error after DLQ route", zap.String("id", raw.ID), zap.Error(ackErr))
			}
			return
		}

		// Under retry threshold — leave in PEL for re-delivery.
		return
	}

	// ACK success.
	if err := q.rdb.XAck(ctx, stream, group, raw.ID).Err(); err != nil {
		q.log.Warn("XACK error", zap.String("id", raw.ID), zap.Error(err))
	}
}
