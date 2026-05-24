package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	PostgresDSN        string
	MongoURI           string
	RedisAddr          string
	RedisPass          string
	JWTSecret          string
	Environment        string
	TrustedProxies     string
	CORSAllowedOrigins string
}

func LoadConfig() *Config {
	// Automatically load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it, relying on system environment variables.")
	}

	return &Config{
		Port:               getEnv("PORT", "8080"),
		PostgresDSN:        getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/vitalcache?sslmode=disable"),
		MongoURI:           getEnv("MONGO_URI", "mongodb://localhost:27017"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:          getEnv("REDIS_PASS", ""),
		JWTSecret:          getEnv("JWT_SECRET", "super-secret-key-change-in-prod"),
		Environment:        getEnv("ENV", "development"),
		TrustedProxies:     getEnv("TRUSTED_PROXIES", "10.0.0.0/8,127.0.0.1/8"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
