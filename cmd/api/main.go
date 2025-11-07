package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/app"
	"github.com/THE-AkS-21/vitalcache-server/internal/observability"
	"github.com/THE-AkS-21/vitalcache-server/internal/queue"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
	"github.com/THE-AkS-21/vitalcache-server/pkg/config"
	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

func main() {
	observability.InitLogger()

	ctx := context.Background()

	// 1) Load secrets
	secClient, err := config.NewSecretsClient(ctx)
	if err != nil {
		slog.Error("aws secrets init failed", "err", err)
		os.Exit(1)
	}
	payload, _, err := secClient.Get(ctx)
	if err != nil {
		slog.Error("get secrets failed", "err", err)
		os.Exit(1)
	}

	// 2) Tracing init (OTLP exporter optional; falls back to no-op)
	if err := observability.InitTracing(ctx); err != nil {
		slog.Warn("tracing init failed", "err", err)
	}
	defer observability.ShutdownTracing(ctx)

	// 3) Metrics init
	observability.InitMetrics()

	// 4) JWT keys + rotation
	keySrc, err := jwt.NewKeySource(payload, func(c context.Context, p *config.SecretPayload) error { return secClient.Put(c, p) })
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

	// 5) DB client
	db := supabase.NewClient(payload.SupabaseURL, payload.SupabaseKey)

	// 6) Queue: start Redis worker (no email, just logs + cleans files)
	q := queue.NewInMemory()
	go func() {
		if err := q.StartWorker(ctx); err != nil {
			slog.Error("queue worker stopped", "err", err)
		}
	}()

	// 7) HTTP server
	r := app.NewServer(db, keySrc, payload, q) // pass queue
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
