package middlewares

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Add trace_id to context for downstream logging
		requestID := c.Writer.Header().Get("X-Request-ID")
		if requestID != "" {
			ctx := context.WithValue(c.Request.Context(), "trace_id", requestID)
			c.Request = c.Request.WithContext(ctx)
		}

		c.Next()

		// Use LogAttrs for zero-allocation structured logging
		latency := time.Since(start)
		slog.LogAttrs(
			c.Request.Context(),
			slog.LevelInfo,
			"http_request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.String("ip", c.ClientIP()),
			slog.Duration("latency", latency),
			slog.Int("bytes", c.Writer.Size()),
		)
	}
}
