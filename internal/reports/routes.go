package reports

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

func RegisterRoutes(router *gin.RouterGroup, pool *pgxpool.Pool, ks pkgjwt.JWTKeySource) {
	repo := NewRepository(pool)
	svc := NewService(repo)
	h := NewHandler(svc)

	g := router.Group("/reports")
	g.Use(middleware.Auth(ks))
	{
		g.GET("/format", h.GetFormat)
		g.PUT("/format", middleware.BlockRole("TESTER"), middleware.RequireRole("Doctor"), h.UpdateFormat)
	}
}
