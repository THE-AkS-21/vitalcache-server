// Package middleware provides structured Zap request logging for VitalCache.
package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
)

// Logger is a Gin middleware that logs every HTTP request as a structured
// Zap entry. It also stores a child logger (with request_id + user_id pre-set)
// in the Gin context so downstream code can call logger.WithContext(ctx).
func Logger() gin.HandlerFunc {
	log := logger.Named("http")

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Attach a request-scoped child logger to the context.
		rid, _ := c.Get("request_id")
		ridStr, _ := rid.(string)
		reqLog := log.With(zap.String("request_id", ridStr))
		logger.WithLogger(c.Request.Context(), reqLog)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// Extract User ID safely from Context
		var userID int64
		if uid, exists := c.Get("user_id"); exists {
			switch v := uid.(type) {
			case float64:
				userID = int64(v)
			case int64:
				userID = v
			case int:
				userID = int64(v)
			case string:
				parsed, _ := strconv.ParseInt(v, 10, 64)
				userID = parsed
			}
		}

		reqLog.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.Int64("user_id", userID),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("bytes", c.Writer.Size()),
		)
	}
}
