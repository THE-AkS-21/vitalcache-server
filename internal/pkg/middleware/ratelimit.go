// Package middleware provides Redis-backed sliding-window rate limiting for VitalCache.
// Unlike the previous in-memory implementation, this works correctly in multi-instance
// deployments and survives server restarts.
//
// Algorithm: Lua-based sliding window using a Redis sorted set per key.
//  1. Remove all entries older than the window.
//  2. Count remaining entries.
//  3. If count < limit, add this request (score=timestamp) and allow.
//  4. Else, deny.
//
// Each key has a TTL pinned to the window size to prevent unbounded key growth.
package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
)

// slidingWindowScript atomically implements the sliding-window algorithm.
// KEYS[1] = sorted set key  (e.g. "rl:auth:192.168.1.1")
// ARGV[1] = window start    (unix timestamp in milliseconds, as string)
// ARGV[2] = now             (unix timestamp in milliseconds, as string)
// ARGV[3] = limit           (max requests allowed in window)
// ARGV[4] = window ms       (window size in milliseconds, used as TTL)
// Returns: "0" if allowed, "1" if denied.
var slidingWindowScript = redis.NewScript(`
local key      = KEYS[1]
local win_start = ARGV[1]
local now       = ARGV[2]
local limit     = tonumber(ARGV[3])
local window_ms = tonumber(ARGV[4])

-- Remove expired entries
redis.call('ZREMRANGEBYSCORE', key, '-inf', win_start)

-- Count current requests in window
local count = redis.call('ZCARD', key)

if count < limit then
  -- Allow: add this request (member = now, score = now)
  redis.call('ZADD', key, now, now)
  -- Reset TTL on each write so keys expire after inactivity
  redis.call('PEXPIRE', key, window_ms)
  return 0
else
  return 1
end
`)

// RateLimiter returns a Gin middleware that enforces a sliding-window rate limit.
//
//	group:     label for the rate-limit bucket (e.g. "auth", "api")
//	rpm:       maximum requests per minute per unique IP
//	rdb:       the Redis client to use
//
// On limit exceeded: HTTP 429 + Retry-After header set to the window size.
func RateLimiter(group string, rpm int, rdb *redis.Client) gin.HandlerFunc {
	windowDur := time.Minute
	windowMs := windowDur.Milliseconds()
	limit := int64(rpm)

	// Circuit Breaker / Fallback: Global in-memory token bucket if Redis goes down.
	// We convert RPM to rate.Limit (requests per second) and use rpm as burst size.
	fallbackRate := rate.Limit(float64(rpm) / 60.0)
	fallbackLimiter := rate.NewLimiter(fallbackRate, rpm)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("rl:%s:%s", group, ip)
		now := time.Now().UnixMilli()
		windowStart := now - windowMs

		result, err := slidingWindowScript.Run(
			c.Request.Context(),
			rdb,
			[]string{key},
			strconv.FormatInt(windowStart, 10),
			strconv.FormatInt(now, 10),
			strconv.FormatInt(limit, 10),
			strconv.FormatInt(windowMs, 10),
		).Int()

		if err != nil {
			// Redis failure — fallback to strict in-memory global token bucket limiter
			// to protect the database from an attack masking as a Redis outage.
			if !fallbackLimiter.Allow() {
				c.Header("Retry-After", "5")
				apperr.Abort(c, apperr.New("RATE_LIMIT_EXCEEDED", "Too many requests. Please try again later."))
				return
			}
			c.Next()
			return
		}

		if result == 1 {
			c.Header("Retry-After", strconv.Itoa(int(windowDur.Seconds())))
			apperr.Abort(c, apperr.New("RATE_LIMIT_EXCEEDED", "Too many requests. Please try again later."))
			return
		}

		c.Next()
	}
}

// RateLimiterFromConfig builds the standard VitalCache rate limiters.
// Returns two middlewares: one for auth routes, one for API routes.
func RateLimiterFromConfig(authRPM, apiRPM int, rdb *redis.Client) (authLimiter, apiLimiter gin.HandlerFunc) {
	return RateLimiter("auth", authRPM, rdb),
		RateLimiter("api", apiRPM, rdb)
}

// noopRateLimiter is a pass-through used when Redis is unavailable.
func noopRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// SafeRateLimiter returns a Redis-backed limiter when rdb != nil,
// otherwise falls back to a no-op (logs a warning).
func SafeRateLimiter(group string, rpm int, rdb *redis.Client) gin.HandlerFunc {
	if rdb == nil {
		return noopRateLimiter()
	}
	return RateLimiter(group, rpm, rdb)
}

// PrometheusRateLimitMiddleware wraps a gin.HandlerFunc to record
// rate-limit hits in Prometheus (wired in Step 8 — Observability).
func withStatusCapture(h gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		h(c)
		// Metrics hook: Status 429 ↦ rate_limit_total{group=...}++
		// Will be wired in the observability step.
		_ = http.StatusTooManyRequests
	}
}
