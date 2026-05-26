// Package config provides the unified application configuration for VitalCache.
// Call Load() once at startup (after godotenv.Load()) to obtain a *Config.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Profile represents the deployment environment.
type Profile string

const (
	ProfileDev     Profile = "dev"
	ProfileStaging Profile = "staging"
	ProfileProd    Profile = "prod"
)

// Config is the single source of truth for all application configuration.
// Every field maps to an environment variable documented in .env.example.
type Config struct {
	Profile  Profile
	Server   ServerConfig
	Postgres PostgresConfig
	Mongo    MongoConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Kafka    KafkaConfig
	CORS     CORSConfig
	Otel     OtelConfig
	Metrics  MetricsConfig
}

// ServerConfig holds HTTP server tunables.
type ServerConfig struct {
	Port              int
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	RateLimit         RateLimitConfig
}

// RateLimitConfig holds per-route Redis sliding-window limits.
type RateLimitConfig struct {
	// AuthRPM: max requests per minute (per IP) on /auth/* routes.
	AuthRPM int
	// APIRPM: max requests per minute (per IP) on /api/v1/* routes.
	APIRPM int
}

// PostgresConfig holds pgx pool settings.
// DSN should be the Supabase direct connection URL:
//
//	postgresql://postgres:[password]@db.[ref].supabase.co:5432/postgres
//
// For Supabase session-mode pooler (recommended for production):
//
//	postgresql://postgres.[ref]:[password]@aws-0-[region].pooler.supabase.com:5432/postgres
type PostgresConfig struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthCheck     time.Duration
}

// MongoConfig targets the existing Atlas cluster.
type MongoConfig struct {
	URI      string
	Database string
}

// RedisConfig supports a full REDIS_URL (preferred, enables TLS via rediss://)
// or discrete REDIS_ADDR + REDIS_PASSWORD as a fallback.
type RedisConfig struct {
	URL      string // e.g. rediss://default:[pw]@host:6379
	Addr     string // fallback: host:port
	Password string // fallback password
	DB       int    // Redis logical DB index (usually 0)
}

// JWTConfig holds signing-key settings.
// The actual key material lives in pkg/jwt (KeySource); only TTLs go here.
type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// KafkaConfig is opt-in; disabled when KAFKA_BROKERS is unset.
type KafkaConfig struct {
	Brokers []string
	Enabled bool
}

// CORSConfig lists allowed origins.
type CORSConfig struct {
	AllowedOrigins []string
}

// OtelConfig for OpenTelemetry tracing export.
type OtelConfig struct {
	Endpoint string
	Enabled  bool
}

// MetricsConfig controls /metrics endpoint visibility.
type MetricsConfig struct {
	Protected bool
	AllowIPs  []string
}

// Load populates Config from environment variables.
// Call after godotenv.Load() so local .env values are visible.
func Load() (*Config, error) {
	profile := Profile(getEnvStr("APP_ENV", "dev"))

	cfg := &Config{
		Profile: profile,
		Server: ServerConfig{
			Port:              getEnvInt("PORT", 8080),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
			RateLimit: RateLimitConfig{
				AuthRPM: getEnvInt("RATE_LIMIT_AUTH_RPM", 5),
				APIRPM:  getEnvInt("RATE_LIMIT_API_RPM", 100),
			},
		},
		Postgres: PostgresConfig{
			DSN:             os.Getenv("POSTGRES_DSN"),
			MaxConns:        int32(getEnvInt("POSTGRES_MAX_CONNS", 25)),
			MinConns:        int32(getEnvInt("POSTGRES_MIN_CONNS", 5)),
			MaxConnLifetime: time.Duration(getEnvInt("POSTGRES_CONN_LIFETIME_MINS", 30)) * time.Minute,
			MaxConnIdleTime: time.Duration(getEnvInt("POSTGRES_CONN_IDLE_MINS", 5)) * time.Minute,
			HealthCheck:     time.Minute,
		},
		Mongo: MongoConfig{
			URI:      os.Getenv("MONGO_URI"),
			Database: getEnvStr("MONGO_DB", "vitalcache"),
		},
		Redis: RedisConfig{
			URL:      os.Getenv("REDIS_URL"),
			Addr:     getEnvStr("REDIS_ADDR", "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          os.Getenv("JWT_SECRET"),
			AccessTokenTTL:  parseDuration(getEnvStr("JWT_EXPIRY", "15m")),
			RefreshTokenTTL: parseDuration(getEnvStr("JWT_REFRESH_EXPIRY", "168h")), // 7d
		},
		Kafka: KafkaConfig{
			Brokers: splitComma(os.Getenv("KAFKA_BROKERS")),
			Enabled: os.Getenv("KAFKA_BROKERS") != "",
		},
		CORS: CORSConfig{
			AllowedOrigins: splitComma(getEnvStr("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		},
		Otel: OtelConfig{
			Endpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
			Enabled:  os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "",
		},
		Metrics: MetricsConfig{
			Protected: os.Getenv("METRICS_PROTECTED") != "",
			AllowIPs:  splitComma(os.Getenv("METRICS_ALLOW_IPS")),
		},
	}

	return cfg, cfg.validate()
}

// IsProd returns true when APP_ENV=prod.
func (c *Config) IsProd() bool { return c.Profile == ProfileProd }

// IsStaging returns true when APP_ENV=staging.
func (c *Config) IsStaging() bool { return c.Profile == ProfileStaging }

// IsDev returns true when APP_ENV=dev (the default).
func (c *Config) IsDev() bool { return c.Profile == ProfileDev }

func (c *Config) validate() error {
	if c.Postgres.DSN == "" {
		return fmt.Errorf("config: POSTGRES_DSN is required")
	}
	if c.Mongo.URI == "" {
		return fmt.Errorf("config: MONGO_URI is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("config: JWT_SECRET is required")
	}
	if c.Redis.URL == "" && c.Redis.Addr == "" {
		return fmt.Errorf("config: one of REDIS_URL or REDIS_ADDR is required")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

func getEnvStr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}
