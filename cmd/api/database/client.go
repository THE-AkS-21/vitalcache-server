package database

import (
	"log/slog"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/config"
	supa "github.com/supabase-community/supabase-go"
)

func NewSupabaseClient(cfg *config.Config) *supa.Client {
	client, err := supa.NewClient(cfg.SupabaseURL, cfg.SupabaseKey, nil)
	if err != nil {
		slog.Error("Failed to initialize Supabase client", "error", err)
		panic(err)
	}
	slog.Info("Supabase client initialized successfully")
	return client
}
