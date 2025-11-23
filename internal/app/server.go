package app

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/app/middlewares"
	"github.com/THE-AkS-21/vitalcache-server/internal/app/routes"
	"github.com/THE-AkS-21/vitalcache-server/internal/infra/kafka"
	"github.com/THE-AkS-21/vitalcache-server/internal/infra/redis"
	"github.com/THE-AkS-21/vitalcache-server/internal/observability"
	"github.com/THE-AkS-21/vitalcache-server/internal/queue"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
	"github.com/THE-AkS-21/vitalcache-server/pkg/config"
	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func NewServer(db *supabase.Client, ks jwt.JWTKeySource, secrets *config.SecretPayload, q queue.Client, rdb *redis.Client, kp *kafka.Producer) *gin.Engine {
	r := gin.New()

	// core middlewares
	r.Use(gin.Recovery())
	r.Use(middlewares.RequestID())
	r.Use(middlewares.Logger())
	r.Use(middlewares.RateLimit(
		float64(getEnvInt("RATE_LIMIT_RPS", 10)),
		int64(getEnvInt("RATE_LIMIT_BURST", 20)),
	))

	// tracing (opt-in)
	r.Use(otelgin.Middleware("vitalcache-api"))

	// metrics
	r.Use(observability.MetricsMiddleware())

	// CORS env-driven: dev=* ; prod = CORS_ALLOWED_ORIGINS
	cc := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "X-Request-ID", "Content-Type"},
		ExposeHeaders:    []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	if gin.Mode() == gin.DebugMode {
		cc.AllowOrigins = []string{"http://localhost:3000", "http://localhost:8080"}
	} else {
		origins := strings.Split(strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")), ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		cc.AllowOrigins = origins
	}
	r.Use(cors.New(cc))

	// health
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/healthz/ready", func(c *gin.Context) {
		// minimal PostgREST ping
		if err := db.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// /metrics (protect in prod)
	if gin.Mode() == gin.DebugMode || os.Getenv("METRICS_PROTECTED") == "" {
		r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	} else {
		allow := strings.Split(strings.TrimSpace(os.Getenv("METRICS_ALLOW_IPS")), ",")
		r.GET("/metrics", func(c *gin.Context) {
			ip := c.ClientIP()
			allowed := false
			for _, a := range allow {
				if ip == strings.TrimSpace(a) && ip != "" {
					allowed = true
					break
				}
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "forbidden", "message": "metrics blocked"}})
				return
			}
			promhttp.Handler().ServeHTTP(c.Writer, c.Request)
		})
	}

	// register routes
	routes.Register(r, db, ks, secrets, q, rdb, kp)
	return r
}

func getEnvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconvAtoi(v); err == nil {
			return n
		}
	}
	return def
}

func strconvAtoi(s string) (int, error) { // tiny inline to avoid pulling strconv in snippet header
	var n int
	_, err := fmtSscanf(s, "%d", &n)
	return n, err
}
func fmtSscanf(s, f string, a ...any) (int, error) { return fmt.Sscanf(s, f, a...) }
