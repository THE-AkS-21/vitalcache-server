package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/app"
	"github.com/THE-AkS-21/vitalcache-server/internal/infra/kafka"
	"github.com/THE-AkS-21/vitalcache-server/internal/infra/redis"
	"github.com/THE-AkS-21/vitalcache-server/internal/observability"
	"github.com/THE-AkS-21/vitalcache-server/internal/queue"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
	"github.com/THE-AkS-21/vitalcache-server/internal/workers"
	"github.com/THE-AkS-21/vitalcache-server/pkg/config"
	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
	"github.com/joho/godotenv"
)

func main() {
	observability.InitLogger()
	_ = godotenv.Load() // load .env early

	ctx := context.Background()

	// 1) Try AWS secrets; if not configured, fall back to .env
	secClient, err := config.NewSecretsClient(ctx)
	if err != nil {
		slog.Error("aws secrets init failed", "err", err)
		os.Exit(1)
	}

	var payload *config.SecretPayload
	var putFunc func(context.Context, *config.SecretPayload) error

	if secClient != nil {
		var ver string
		payload, ver, err = secClient.Get(ctx)
		if err != nil {
			slog.Error("get secrets failed", "err", err)
			os.Exit(1)
		}
		slog.Info("loaded secrets from AWS", "version", ver)
		putFunc = secClient.Put
	} else {
		// Local fallback: read Supabase creds from env, use a dev keyring
		url := os.Getenv("SUPABASE_URL")
		key := os.Getenv("SUPABASE_KEY")
		if url == "" || key == "" {
			slog.Error("missing SUPABASE_URL or SUPABASE_KEY in .env for local fallback")
			os.Exit(1)
		}
		payload = &config.SecretPayload{
			SupabaseURL: url,
			SupabaseKey: key,
			JWTKeyring:  config.NewLocalKeyring(),
		}
		// No-op put function in dev
		putFunc = func(context.Context, *config.SecretPayload) error { return nil }
		slog.Warn("using local .env secrets (AWS disabled)")
	}

	// 2) Tracing + metrics
	if err := observability.InitTracing(ctx); err != nil {
		slog.Warn("tracing init failed", "err", err)
	}
	defer observability.ShutdownTracing(ctx)
	observability.InitMetrics()

	// 3) JWT keys + rotation (Rotation will no-op in local fallback)
	keySrc, err := jwt.NewKeySource(payload, putFunc)
	if err != nil {
		slog.Error("jwt key source failed", "err", err)
		os.Exit(1)
	}
	if err := jwt.RotateIfNeeded(ctx, keySrc); err != nil {
		slog.Error("jwt rotate check failed", "err", err)
	}
	go func() {
		t := time.NewTicker(24 * time.Hour)
		defer t.Stop()
		for range t.C {
			if err := jwt.RotateIfNeeded(context.Background(), keySrc); err != nil {
				slog.Error("daily jwt rotate failed", "err", err)
			}
		}
	}()

	// 4) DB client
	db := supabase.NewClient(payload.SupabaseURL, payload.SupabaseKey)

	// 5) Redis & Queue
	rdb, err := redis.NewClient(ctx)
	if err != nil {
		slog.Warn("redis init failed, falling back to in-memory queue", "err", err)
	}

	var q queue.Client
	if rdb != nil {
		q = queue.NewRedisQueue(rdb.Client)
	} else {
		q = queue.NewInMemory()
	}

	go func() {
		if err := q.StartWorker(ctx); err != nil {
			slog.Error("queue worker stopped", "err", err)
		}
	}()

	// Start Email Worker (Stubbed)
	emailWorker := workers.NewEmailWorker(q)
	emailWorker.Start(ctx)

	// 6) Kafka
	kp, err := kafka.NewProducer()
	if err != nil {
		slog.Warn("kafka init failed", "err", err)
	}
	if kp != nil {
		defer kp.Close()
	}

	// 7) HTTP server
	r := app.NewServer(db, keySrc, payload, q, rdb, kp)
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("vitalcache server starting", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
		slog.Error("server error", "err", err)
	}
}
