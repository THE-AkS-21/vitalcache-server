package main

import (
	"context"
	"log"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/worker"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/THE-AkS-21/vitalcache-server/internal/appointments"
	"github.com/THE-AkS-21/vitalcache-server/internal/auth"
	"github.com/THE-AkS-21/vitalcache-server/internal/config"
	"github.com/THE-AkS-21/vitalcache-server/internal/medical_reports"
	"github.com/THE-AkS-21/vitalcache-server/internal/medicines"
	"github.com/THE-AkS-21/vitalcache-server/internal/observability"
	"github.com/THE-AkS-21/vitalcache-server/internal/patients"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/db"
	"github.com/THE-AkS-21/vitalcache-server/internal/prescriptions"
	"github.com/THE-AkS-21/vitalcache-server/internal/reports"
)

func main() {
	cfg := config.LoadConfig()

	// --- 1. Observability ---
	tp, err := observability.InitTracer(cfg.Environment)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer: %v", err)
		}
	}()

	// --- 2. Infrastructure ---
	pgPool, err := db.NewPostgresPool(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer pgPool.Close()

	mongoClient, err := db.NewMongoClient(cfg.MongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to Mongo: %v", err)
	}

	redisClient, err := db.NewRedisClient(cfg.RedisAddr, cfg.RedisPass)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	jobProcessor := worker.NewProcessor(redisClient)
	jobProcessor.Start(workerCtx)

	// --- 3. API Router Setup ---
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	// Apply Observability Middleware globally
	router.Use(observability.MetricsMiddleware())

	// Expose Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/api/v1")

	v1.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "VitalCache API is online"})
	})

	// --- 4. Register Domains ---
	ks := auth.RegisterRoutes(v1, pgPool, redisClient, cfg.JWTSecret)
	appointments.RegisterRoutes(v1, pgPool, redisClient, ks)
	medicines.RegisterRoutes(v1, mongoClient, redisClient, ks)
	patients.RegisterRoutes(v1, pgPool, ks)
	prescriptions.RegisterRoutes(v1, mongoClient, redisClient, ks)
	reports.RegisterRoutes(v1, pgPool, ks)
	medical_reports.RegisterRoutes(v1, mongoClient, ks)

	// --- 5. Start Server ---
	log.Printf("Server starting on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
