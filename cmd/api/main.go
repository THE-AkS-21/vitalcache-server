package main

import (
	"log/slog"
	"os"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/config"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/database"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/logging"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/routes"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/services" // Import services
	"github.com/gin-gonic/gin"
)

func main() {
	logging.InitLogger()
	cfg := config.LoadConfig()
	db := database.NewSupabaseClient(cfg)

	// Initialize the email queue and start the worker
	services.InitEmailQueue()
	services.StartEmailWorker()

	if cfg.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := routes.SetupRouter(db, cfg)

	slog.Info("🚀 Starting server", "port", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		slog.Error("❌ Failed to start server", "error", err)
		os.Exit(1)
	}
}
