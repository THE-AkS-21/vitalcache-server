package app

import (
	"github.com/THE-AkS-21/vitalcache-server/internal/app/middlewares"
	"github.com/THE-AkS-21/vitalcache-server/internal/app/routes"
	"github.com/THE-AkS-21/vitalcache-server/internal/observability"
	"github.com/THE-AkS-21/vitalcache-server/internal/queue"
	"github.com/THE-AkS-21/vitalcache-server/pkg/config"
	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	supa "github.com/supabase-community/supabase-go"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func NewServer(db *supa.Client, ks jwt.JWTKeySource, secrets *config.SecretPayload, q queue.Client) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middlewares.RequestID())
	r.Use(otelgin.Middleware("vitalcache-api")) // tracing
	r.Use(observability.MetricsMiddleware())    // metrics
	r.Use(middlewares.Logger())
	r.Use(middlewares.RateLimit(10, 20)) // 10 rps, burst 20

	cc := cors.DefaultConfig()
	cc.AllowHeaders = append(cc.AllowHeaders, "Authorization", "X-Request-ID")
	cc.AllowAllOrigins = true // tighten in prod
	r.Use(cors.New(cc))

	// /metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	routes.Register(r, db, ks, secrets, q)
	return r
}
