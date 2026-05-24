package appointments

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

func RegisterRoutes(router *gin.RouterGroup, db *pgxpool.Pool, rdb *redis.Client, ks pkgjwt.JWTKeySource) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	idemMiddleware := middleware.Idempotency(rdb, 24*time.Hour)

	g := router.Group("/appointments")
	g.Use(middleware.Auth(ks))
	{
		// Both doctors and staff can list/create appointments
		g.GET("/", h.List)
		g.POST("/", idemMiddleware, middleware.RequireRole("DOCTOR", "HOSPITAL_STAFF", "DEVELOPER"), h.Create)
	}
}
