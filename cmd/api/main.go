package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/THE-AkS-21/vitalcache-server/internal/config"

	// Core Packages
	"github.com/THE-AkS-21/vitalcache-server/internal/observability"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/db"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	"github.com/THE-AkS-21/vitalcache-server/internal/worker"

	// Domain Modules
	"github.com/THE-AkS-21/vitalcache-server/internal/appointments"
	"github.com/THE-AkS-21/vitalcache-server/internal/auth"
	"github.com/THE-AkS-21/vitalcache-server/internal/billings"
	"github.com/THE-AkS-21/vitalcache-server/internal/doctors"
	"github.com/THE-AkS-21/vitalcache-server/internal/invites"
	"github.com/THE-AkS-21/vitalcache-server/internal/medical_reports"
	"github.com/THE-AkS-21/vitalcache-server/internal/medicines"
	"github.com/THE-AkS-21/vitalcache-server/internal/patients"
	"github.com/THE-AkS-21/vitalcache-server/internal/prescriptions"
	"github.com/THE-AkS-21/vitalcache-server/internal/reports"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.LoadConfig()

	isDev := cfg.Environment != "production"
	logger.Init(isDev)
	defer logger.Sync()

	logger.L().Info("Starting VitalCache Server...")

	// --- 1. Observability ---
	tp, err := observability.InitTracer(cfg.Environment)
	if err != nil {
		logger.L().Sugar().Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			logger.L().Sugar().Errorf("Error shutting down tracer: %v", err)
		}
	}()

	// --- 2. Infrastructure ---
	pgPool, err := db.NewPostgresPool(cfg.PostgresDSN)
	if err != nil {
		logger.L().Sugar().Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	mongoClient, err := db.NewMongoClient(cfg.MongoURI)
	if err != nil {
		logger.L().Sugar().Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background()) //nolint:errcheck

	redisClient, err := db.NewRedisClient(cfg.RedisAddr, cfg.RedisPass)
	if err != nil {
		logger.L().Sugar().Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Start background job processor
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	jobProcessor := worker.NewProcessor(redisClient)
	jobProcessor.Start(workerCtx)

	outboxPoller := worker.NewOutboxPoller(redisClient, mongoClient.Database("vitalcache"))
	outboxPoller.Start(workerCtx)

	worker.StartBillingAggregator(workerCtx, pgPool, logger.L().Sugar())

	// --- 3. Router Setup ---
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	if cfg.TrustedProxies != "" {
		_ = router.SetTrustedProxies(strings.Split(cfg.TrustedProxies, ","))
	} else {
		_ = router.SetTrustedProxies(nil)
	}

	// ── Global middleware ──────────────────────────────────────────────────────
	sugarLog := logger.L().Sugar()

	// MaxBytesReader middleware to prevent memory exhaustion DoS
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20) // 2MB limit
		c.Next()
	})

	router.Use(middleware.Recovery(sugarLog))
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	router.Use(observability.MetricsMiddleware())

	// ── Observability endpoints ────────────────────────────────────────────────
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"version": "v1",
		})
	})

	v1 := router.Group("/api/v1")

	// ── Rate limiting (Redis-backed sliding window) ────────────────────────────
	authLimiter, apiLimiter := middleware.RateLimiterFromConfig(60, 300, redisClient)
	v1.Use(apiLimiter)

	// ── Domain Routes ──────────────────────────────────────────────────────────
	// auth.RegisterRoutes returns the JWTKeySource used by all other modules.
	ks := auth.RegisterRoutes(v1, pgPool, redisClient, cfg.JWTSecret)

	// Apply stricter rate limit to auth routes
	v1.Group("/auth").Use(authLimiter)

	doctors.RegisterRoutes(v1, pgPool, nil, ks)
	patients.RegisterRoutes(v1, pgPool, ks)
	appointments.RegisterRoutes(v1, pgPool, redisClient, ks)
	medicines.RegisterRoutes(v1, mongoClient, redisClient, ks)
	prescriptions.RegisterRoutes(v1, mongoClient, redisClient, ks)
	reports.RegisterRoutes(v1, pgPool, ks)
	medical_reports.RegisterRoutes(v1, mongoClient, ks)
	billings.RegisterRoutes(v1, pgPool, ks)
	invites.RegisterRoutes(v1, pgPool, ks)

	v1.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "VitalCache API is online"})
	})

	// --- 4. Start Server with Graceful Shutdown ---
	port := cfg.Port
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.L().Sugar().Infof("Server listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L().Sugar().Fatalf("listen: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.L().Info("Shutdown signal received, gracefully terminating...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.L().Sugar().Fatalf("Server forced to shutdown: %v", err)
	}

	// Give workers a chance to finish in-flight jobs
	workerCancel()
	logger.L().Info("Waiting for background jobs to complete...")
	jobProcessor.Wait()

	logger.L().Info("Server exiting gracefully")
}
