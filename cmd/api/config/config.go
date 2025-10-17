package config

import (
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	GinMode     string
	SupabaseURL string
	SupabaseKey string
	JWTSecret   string
}

func LoadConfig() *Config {
	// Load .env file only if not in production
	if os.Getenv("GIN_MODE") != "release" {
		err := godotenv.Load()
		if err != nil {
			slog.Warn("Could not load .env file, relying on OS environment variables")
		}
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		GinMode:     getEnv("GIN_MODE", "debug"),
		SupabaseURL: getEnv("SUPABASE_URL", ""),
		SupabaseKey: getEnv("SUPABASE_KEY", ""),
		JWTSecret:   getEnv("JWT_SECRET", ""),
	}
}

// Helper to read an environment variable or panic if not present
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if fallback == "" {
		log.Fatalf("FATAL: Required environment variable %s is not set.", key)
	}
	return fallback
}
