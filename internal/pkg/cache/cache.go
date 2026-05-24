// Package cache provides a Redis-backed cache-aside helper for VitalCache.
//
// Cache-aside pattern:
//  1. Check Redis for the key.
//  2. On hit: deserialise and return.
//  3. On miss: call the load function, store the result in Redis with the given TTL, and return.
//
// Default TTLs:
//
//	medicine catalog → 15 minutes
//	doctor list      → 10 minutes
//	prescription     →  5 minutes
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
)

// Well-known TTLs — use these constants so all repos stay consistent.
const (
	TTLMedicines    = 15 * time.Minute
	TTLDoctors      = 10 * time.Minute
	TTLPrescription = 5 * time.Minute
	TTLProfile      = 10 * time.Minute
)

// Cache is a thin Redis-backed JSON cache.
type Cache struct {
	rdb *redis.Client
	log *zap.Logger
	sfg singleflight.Group
}

// New returns a Cache backed by the provided Redis client.
func New(rdb *redis.Client) *Cache {
	return &Cache{
		rdb: rdb,
		log: logger.Named("cache"),
	}
}

// Get deserialises the cached value at key into dest.
// Returns redis.Nil if the key does not exist (cache miss).
func (c *Cache) Get(ctx context.Context, key string, dest any) error {
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return err // callers check for redis.Nil
	}
	return json.Unmarshal(data, dest)
}

// Set serialises val as JSON and stores it in Redis with the given TTL.
func (c *Cache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("cache.Set marshal: %w", err)
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

// Del removes the key from Redis.
func (c *Cache) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// GetOrLoad implements cache-aside using Generics and singleflight to prevent
// Thundering Herd (cache stampedes).
//
//	medicines, err := cache.GetOrLoad(ctx, c, "medicines:page:0", cache.TTLMedicines,
//	    func() ([]domain.Medicine, error) { return repo.ListMedicines(ctx, 20, 0) },
//	)
func GetOrLoad[T any](ctx context.Context, c *Cache, key string, ttl time.Duration, loadFn func() (T, error)) (T, error) {
	var dest T
	err := c.Get(ctx, key, &dest)
	if err == nil {
		c.log.Debug("cache hit", zap.String("key", key))
		return dest, nil
	}
	if !errors.Is(err, redis.Nil) {
		// Redis error — log and fall through to loadFn so the request still works.
		c.log.Warn("cache get error, falling through to load", zap.String("key", key), zap.Error(err))
	} else {
		c.log.Debug("cache miss", zap.String("key", key))
	}

	val, err, _ := c.sfg.Do(key, func() (interface{}, error) {
		loadedVal, loadErr := loadFn()
		if loadErr != nil {
			return loadedVal, loadErr
		}

		// Directly cache the value without double unmarshaling into a pointer
		data, marshalErr := json.Marshal(loadedVal)
		if marshalErr != nil {
			return loadedVal, fmt.Errorf("cache.GetOrLoad marshal: %w", marshalErr)
		}

		// Best-effort write
		if setErr := c.rdb.Set(ctx, key, data, ttl).Err(); setErr != nil {
			c.log.Warn("cache set error", zap.String("key", key), zap.Error(setErr))
		}
		return loadedVal, nil
	})

	if err != nil {
		var zero T
		return zero, err
	}

	return val.(T), nil
}

// Invalidate removes one or more cache keys (e.g. after a write operation).
func (c *Cache) Invalidate(ctx context.Context, keys ...string) {
	if err := c.Del(ctx, keys...); err != nil {
		c.log.Warn("cache invalidate error", zap.Strings("keys", keys), zap.Error(err))
	}
}

// KeyMedicines returns the standard Redis key for a paginated medicine list page.
func KeyMedicines(query string, limit, offset int) string {
	return fmt.Sprintf("medicines:q=%s:l=%d:o=%d", query, limit, offset)
}

// KeyPrescription returns the Redis key for a single prescription document.
func KeyPrescription(id string) string {
	return fmt.Sprintf("prescription:%s", id)
}

// KeyDoctorList returns the Redis key for a paginated doctor list.
func KeyDoctorList(limit, offset int) string {
	return fmt.Sprintf("doctors:l=%d:o=%d", limit, offset)
}

// KeyProfile returns the Redis key for a user's profile.
func KeyProfile(userID int64) string {
	return fmt.Sprintf("profile:%d", userID)
}
