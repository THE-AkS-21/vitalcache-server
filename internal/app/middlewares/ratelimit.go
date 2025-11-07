package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
)

type bucketEntry struct {
	b *ratelimit.Bucket
	t time.Time
}

var (
	ipBuckets = make(map[string]*bucketEntry)
	mu        sync.Mutex
)

// RateLimit fixes the earlier bug: 10 rps truly means 10 requests/second
func RateLimit(rps float64, burst int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		be, ok := ipBuckets[ip]
		if !ok {
			be = &bucketEntry{
				b: ratelimit.NewBucketWithRate(rps, burst),
				t: now,
			}
			ipBuckets[ip] = be
		}
		be.t = now
		mu.Unlock()

		if be.b.TakeAvailable(1) == 0 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}
		c.Next()

		// prune old buckets occasionally
		if now.Unix()%60 == 0 {
			mu.Lock()
			for k, v := range ipBuckets {
				if now.Sub(v.t) > 10*time.Minute {
					delete(ipBuckets, k)
				}
			}
			mu.Unlock()
		}
	}
}
