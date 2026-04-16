package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type cachedResponse struct {
	Status int         `json:"status"`
	Body   interface{} `json:"body"`
}

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// Idempotency ensures POST/PATCH requests are not processed twice
func Idempotency(rdb *redis.Client, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		idemKey := c.GetHeader("Idempotency-Key")
		if idemKey == "" {
			c.Next()
			return
		}

		// User-specific prefix to prevent cross-user key collisions
		userID, _ := c.Get("user_id")
		redisKey := "idempotency:" + idemKey
		if userID != nil {
			redisKey += ":" + userID.(string)
		}

		// 1. Check if response is already cached
		ctx := context.Background()
		cachedData, err := rdb.Get(ctx, redisKey).Result()
		if err == nil && cachedData != "" {
			var resp cachedResponse
			if json.Unmarshal([]byte(cachedData), &resp) == nil {
				c.Header("X-Idempotent-Replayed", "true")
				c.JSON(resp.Status, resp.Body)
				c.Abort()
				return
			}
		}

		// 2. Lock the key (prevent concurrent identical requests)
		locked, err := rdb.SetNX(ctx, redisKey, "PROCESSING", ttl).Result()
		if err != nil || !locked {
			apperr.Abort(c, apperr.Conflict("Request already processing", nil))
			return
		}

		// 3. Intercept the response
		w := &responseBodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		// 4. Cache the successful response
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			var jsonBody interface{}
			if err := json.Unmarshal(w.body.Bytes(), &jsonBody); err == nil {
				respToCache := cachedResponse{
					Status: c.Writer.Status(),
					Body:   jsonBody,
				}
				cacheBytes, _ := json.Marshal(respToCache)
				rdb.Set(ctx, redisKey, cacheBytes, ttl)
			}
		} else {
			// If request failed, delete the lock so they can retry
			rdb.Del(ctx, redisKey)
		}
	}
}
