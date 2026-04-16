package appointments

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
)

func RegisterRoutes(router *gin.RouterGroup, db *pgxpool.Pool, rdb *redis.Client) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	// Idempotency cached for 24 hours
	idemMiddleware := middleware.Idempotency(rdb, 24*time.Hour)

	// Create group. Auth middleware should be applied here or globally above this
	appointmentsGroup := router.Group("/appointments")
	{
		// Idempotency applied ONLY to POST requests
		appointmentsGroup.POST("/", idemMiddleware, h.Create)
		appointmentsGroup.GET("/", h.List)
	}
}
