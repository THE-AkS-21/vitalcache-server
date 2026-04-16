package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	// FIX: Corrected config import path
	"github.com/THE-AkS-21/vitalcache-server/pkg/config"

	// Core Packages
	"github.com/THE-AkS-21/vitalcache-server/internal/observability"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/cache"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/db"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"

	// Domain Modules
	"github.com/THE-AkS-21/vitalcache-server/internal/appointments"
	"github.com/THE-AkS-21/vitalcache-server/internal/auth"
	"github.com/THE-AkS-21/vitalcache-server/internal/doctors"
	"github.com/THE-AkS-21/vitalcache-server/internal/medicines"
	"github.com/THE-AkS-21/vitalcache-server/internal/patients"
	"github.com/THE-AkS-21/vitalcache-server/internal/prescriptions"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)
	defer log.Sync()

	log.Info("Starting VitalCache Server (Modular Monolith)...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	observability.InitTracer(cfg)
	observability.InitMetrics()

	pgPool, err := db.NewPostgresPool(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	mongoClient, err := db.NewMongoClient(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	redisClient, err := cache.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// Ensure these middleware functions exist in internal/pkg/middleware/
	// (You created them in previous steps)
	router.Use(middleware.Recovery(log))
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(log))
	router.Use(middleware.CORS(cfg.FrontendURL))
	router.Use(observability.MetricsMiddleware())

	v1 := router.Group("/api/v1")

	auth.RegisterRoutes(v1, pgPool, redisClient, cfg, log)
	doctors.RegisterRoutes(v1, pgPool, log)
	patients.RegisterRoutes(v1, pgPool, mongoClient, log)
	appointments.RegisterRoutes(v1, pgPool, redisClient, log)
	medicines.RegisterRoutes(v1, mongoClient, redisClient, log)
	prescriptions.RegisterRoutes(v1, mongoClient, redisClient, log)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "postgres": "up", "mongo": "up", "redis": "up"})
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Infof("Server listening on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutdown signal received, gracefully terminating...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Server exiting gracefully")
}
