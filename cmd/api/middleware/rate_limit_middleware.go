package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
)

var (
	ips = make(map[string]*ratelimit.Bucket)
	mu  sync.Mutex
)

// RateLimitMiddleware creates a new bucket for each IP address that limits requests.
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		mu.Lock()
		defer mu.Unlock()

		ip := c.ClientIP()
		if _, ok := ips[ip]; !ok {
			// Rate: 10 requests per second, with a burst capacity of 20
			ips[ip] = ratelimit.NewBucket(10*time.Second, 20)
		}

		if ips[ip].TakeAvailable(1) == 0 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}

		c.Next()
	}
}
