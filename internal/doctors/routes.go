package doctors

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func RegisterRoutes(router *gin.RouterGroup, pgPool *pgxpool.Pool, log *zap.SugaredLogger) {
	repository := NewPgxRepo(pgPool)
	service := NewService(repository, log)
	handler := NewHandler(service, log)

	group := router.Group("/doctors")
	{
		group.GET("", handler.List)
		group.GET("/:id", handler.GetByID)
	}
}
