package billings

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

	group := router.Group("/billings")
	group.Use(middleware.Auth(ks))
	{
		// All authenticated users can list (filtered by their role in handler)
		group.GET("", h.List)

		// Only Doctors/Staff can create billing (when checking out patient)
		group.POST("", middleware.BlockRole("TESTER"), middleware.RequireRole("DOCTOR", "ADMIN", "DEVELOPER"), h.Create)

		// Updating status (e.g. marking as PAID)
		group.PUT("/:id/status", middleware.BlockRole("TESTER"), middleware.RequireRole("DOCTOR", "ADMIN", "DEVELOPER"), h.UpdateStatus)

		// Analytics endpoints for Doctors
		group.GET("/analytics", middleware.RequireRole("DOCTOR"), h.GetAnalytics)
		group.GET("/heatmap", middleware.RequireRole("DOCTOR"), h.GetHeatmap)
	}
}
