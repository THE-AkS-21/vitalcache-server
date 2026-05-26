package medical_reports

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

func RegisterRoutes(router *gin.RouterGroup, mongoClient *mongo.Client, ks pkgjwt.JWTKeySource) {
	db := mongoClient.Database("vitalcache")
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	g := router.Group("/medical-reports")
	g.Use(middleware.Auth(ks))
	{
		g.POST("/", middleware.BlockRole("TESTER"), middleware.RequireRole("Doctor"), h.Create)
		g.GET("/:id", h.Get)
		g.GET("/patient/:id", h.PatientHistory)
	}
}
