package main

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/supabase-community/supabase-go"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/config"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/database"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/logging"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/routes"
)

func main() {
	logging.InitLogger()

	cfg := config.LoadConfig()
	db := database.NewSupabaseClient(cfg)

	app := &App{
		Config: cfg,
		DB:     db,
	}

	app.Start()
}

type App struct {
	Config *config.Config
	DB     *supabase.Client
}

func (a *App) Start() {
	if a.Config.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := routes.SetupRouter(a.DB, a.Config)
	slog.Info("🚀 Starting server", "port", a.Config.Port)

	if err := router.Run(":" + a.Config.Port); err != nil {
		slog.Error("❌ Failed to start server", "error", err)
		os.Exit(1)
	}
}
