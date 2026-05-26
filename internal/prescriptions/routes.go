package prescriptions

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

func RegisterRoutes(router *gin.RouterGroup, mongoClient *mongo.Client, rdb *redis.Client, ks pkgjwt.JWTKeySource) {
	db := mongoClient.Database("vitalcache")
	repo := NewRepository(db)
	svc := NewService(repo, rdb)
	h := NewHandler(svc)

	idemMiddleware := middleware.Idempotency(rdb, 24*time.Hour)

	g := router.Group("/prescriptions")
	g.Use(middleware.Auth(ks))
	{
		// Only doctors can create prescriptions (handler also validates doctor_id claim)
		g.POST("/", middleware.BlockRole("TESTER"), middleware.RequireRole("DOCTOR", "DEVELOPER"), idemMiddleware, h.Create)
		// Any authenticated user can view a prescription (further ownership check in service layer)
		g.GET("/:id", h.Get)
		g.GET("/patient/:patientId", h.ListPatientHistory)
	}
}
