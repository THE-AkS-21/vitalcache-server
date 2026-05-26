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
	body        *bytes.Buffer
	overflow    bool
	maxBodySize int
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	if !r.overflow {
		if r.body.Len()+len(b) > r.maxBodySize {
			r.overflow = true
			r.body.Reset() // Clear what we've buffered to save memory
		} else {
			r.body.Write(b)
		}
	}
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

		ctx := context.Background()

		// 1. Lock the key BEFORE checking (prevents race condition)
		locked, err := rdb.SetNX(ctx, redisKey, "PROCESSING", ttl).Result()
		if err != nil {
			apperr.Abort(c, apperr.Internal(err))
			return
		}

		if !locked {
			// Lock not acquired: it's either PROCESSING or successfully cached
			val, err := rdb.Get(ctx, redisKey).Result()
			if err != nil || val == "PROCESSING" {
				apperr.Abort(c, apperr.Conflict("Request already processing", nil))
				return
			}

			// It's a cached response
			var resp cachedResponse
			if json.Unmarshal([]byte(val), &resp) == nil {
				c.Header("X-Idempotent-Replayed", "true")
				c.JSON(resp.Status, resp.Body)
				c.Abort()
				return
			}

			apperr.Abort(c, apperr.Conflict("Request already processing", nil))
			return
		}

		// 3. Intercept the response with a memory limit
		w := &responseBodyWriter{
			body:           bytes.NewBufferString(""),
			ResponseWriter: c.Writer,
			maxBodySize:    2 * 1024 * 1024, // 2MB limit
		}
		c.Writer = w

		c.Next()

		// 4. Cache the successful response
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			if w.overflow {
				// Too large to cache safely. Delete lock so it doesn't block future requests,
				// though this sacrifices strict idempotency for this specific massive response.
				rdb.Del(ctx, redisKey)
			} else {
				var jsonBody interface{}
				if err := json.Unmarshal(w.body.Bytes(), &jsonBody); err == nil {
					respToCache := cachedResponse{
						Status: c.Writer.Status(),
						Body:   jsonBody,
					}
					cacheBytes, _ := json.Marshal(respToCache)
					rdb.Set(ctx, redisKey, cacheBytes, ttl)
				} else {
					rdb.Del(ctx, redisKey)
				}
			}
		} else {
			// If request failed, delete the lock so they can retry
			rdb.Del(ctx, redisKey)
		}
	}
}
