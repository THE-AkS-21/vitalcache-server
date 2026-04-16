package patients

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(router *gin.RouterGroup, db *pgxpool.Pool) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	patientsGroup := router.Group("/patients")
	{
		// Creation is now handled by POST /auth/register
		patientsGroup.GET("/", h.List)
	}
}
