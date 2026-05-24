package worker

import (
	"context"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type OutboxPoller struct {
	rdb        *redis.Client
	outboxColl *mongo.Collection
	log        *zap.Logger
}

func NewOutboxPoller(rdb *redis.Client, db *mongo.Database) *OutboxPoller {
	return &OutboxPoller{
		rdb:        rdb,
		outboxColl: db.Collection("outbox"),
		log:        logger.Named("outbox_poller"),
	}
}

func (p *OutboxPoller) Start(ctx context.Context) {
	p.log.Info("Starting MongoDB outbox poller...")

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				p.log.Info("Shutting down outbox poller...")
				return
			case <-ticker.C:
				p.pollAndForward(ctx)
			}
		}
	}()
}

func (p *OutboxPoller) pollAndForward(ctx context.Context) {
	// Find pending events
	filter := bson.M{"status": "PENDING"}
	cursor, err := p.outboxColl.Find(ctx, filter)
	if err != nil {
		p.log.Error("Failed to query outbox", zap.Error(err))
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var event struct {
			ID      interface{} `bson:"_id"`
			Type    string      `bson:"type"` // e.g. "vitalcache:jobs"
			Payload string      `bson:"payload"`
		}
		if err := cursor.Decode(&event); err != nil {
			p.log.Error("Failed to decode event", zap.Error(err))
			continue
		}

		// Forward to Redis
		if err := p.rdb.RPush(ctx, event.Type, []byte(event.Payload)).Err(); err != nil {
			p.log.Error("Failed to push to Redis", zap.Error(err))
			continue
		}

		// Mark as DONE (or delete)
		_, err = p.outboxColl.DeleteOne(ctx, bson.M{"_id": event.ID})
		if err != nil {
			p.log.Error("Failed to delete processed event", zap.Error(err))
		}
	}
}
