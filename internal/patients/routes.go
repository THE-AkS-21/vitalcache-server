package patients

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

func RegisterRoutes(router *gin.RouterGroup, db *pgxpool.Pool, ks pkgjwt.JWTKeySource) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	g := router.Group("/patients")
	g.Use(middleware.Auth(ks))
	g.Use(middleware.RequireRole("DOCTOR", "HOSPITAL_STAFF", "DEVELOPER"))
	{
		g.GET("/", h.List)
		g.GET("/search", h.Search)
		g.POST("/", h.Create)
		g.GET("/:id", h.GetByID)
		g.PATCH("/:id", h.Update)
	}
}
