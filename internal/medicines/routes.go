package medicines

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

func RegisterRoutes(router *gin.RouterGroup, mongoClient *mongo.Client, rdb *redis.Client) {
	db := mongoClient.Database("vitalcache")
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, rdb)

	medicinesGroup := router.Group("/medicines")
	{
		medicinesGroup.GET("/search", h.Search)
	}
}
