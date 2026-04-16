package prescriptions

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
)

func RegisterRoutes(router *gin.RouterGroup, mongoClient *mongo.Client, rdb *redis.Client) {
	// Select the specific database
	db := mongoClient.Database("vitalcache")

	repo := NewRepository(db)

	// ✅ FIXED: Pass rdb to the service so it can trigger background jobs
	svc := NewService(repo, rdb)

	h := NewHandler(svc)

	idemMiddleware := middleware.Idempotency(rdb, 24*time.Hour)

	prescriptionsGroup := router.Group("/prescriptions")
	{
		prescriptionsGroup.POST("/", idemMiddleware, h.Create)
		prescriptionsGroup.GET("/:id", h.Get)
		prescriptionsGroup.GET("/patient/:patientId", h.ListPatientHistory)
	}
}
