package supabase

import (
	"log/slog"

	supa "github.com/supabase-community/supabase-go"
)

func NewClient(url, key string) *supa.Client {
	client, err := supa.NewClient(url, key, nil)
	if err != nil {
		slog.Error("supabase client init failed", "err", err)
		panic(err)
	}
	slog.Info("supabase client initialized")
	return client
}
