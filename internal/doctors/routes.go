package doctors

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

func RegisterRoutes(router *gin.RouterGroup, pgPool *pgxpool.Pool, _ *zap.SugaredLogger, ks pkgjwt.JWTKeySource) {
	repo := New(pgPool)
	svc := NewService(repo)
	h := NewHandler(svc)

	group := router.Group("/doctors")
	group.Use(middleware.Auth(ks))
	{
		// Any authenticated user can browse doctors (patients need to find their doctor)
		group.GET("", h.List)
		group.GET("/:id", h.GetByID)
	}
}
