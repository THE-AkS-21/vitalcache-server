package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	supabase "github.com/supabase-community/supabase-go"
)

type Config struct {
	Port           string
	GinMode        string
	SupabaseURL    string
	SupabaseKey    string
	JWTSecret      string
	SupabaseClient *supabase.Client
}

func LoadConfig() *Config {
	// Load .env only in non-production
	if os.Getenv("GIN_MODE") != "release" {
		if err := godotenv.Load(); err != nil {
			slog.Warn("Could not load .env file, relying on OS environment variables")
		}
	}

	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		GinMode:     getEnv("GIN_MODE", "debug"),
		SupabaseURL: getEnv("SUPABASE_URL", ""),
		SupabaseKey: getEnv("SUPABASE_KEY", ""),
		JWTSecret:   getEnv("JWT_SECRET", ""),
	}

	// Initialize Supabase client
	client, _ := supabase.NewClient(cfg.SupabaseURL, cfg.SupabaseKey, nil)
	cfg.SupabaseClient = client

	slog.Info("✅ Configuration loaded successfully")
	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if fallback == "" {
		slog.Error("FATAL: Required environment variable is missing", "key", key)
		os.Exit(1)
	}
	return fallback
}
