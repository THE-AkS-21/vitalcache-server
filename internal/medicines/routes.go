package medicines

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

func RegisterRoutes(router *gin.RouterGroup, mongoClient *mongo.Client, rdb *redis.Client, ks pkgjwt.JWTKeySource) {
	db := mongoClient.Database("vitalcache")
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, rdb)

	g := router.Group("/medicines")
	g.Use(middleware.Auth(ks))
	{
		// Search is available to all authenticated users
		g.GET("/search", h.Search)
		g.GET("/", h.Search) // alias for list

		// Only doctors can create new medicines
		g.POST("/", middleware.BlockRole("TESTER"), middleware.RequireRole("Doctor"), h.Create)
	}
}
