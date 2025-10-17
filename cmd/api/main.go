package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/config"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/database"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/logging"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/routes"
)

func main() {
	// Initialize logger first
	logging.InitLogger()

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database client
	db := database.NewSupabaseClient(cfg)

	// Set Gin mode for production
	if cfg.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Setup router
	router := routes.SetupRouter(db, cfg)

	slog.Info(fmt.Sprintf("Starting server on port %s", cfg.Port))
	if err := router.Run(":" + cfg.Port); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
