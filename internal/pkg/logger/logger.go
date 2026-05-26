// Package logger provides a structured, levelled logger for VitalCache
// based on go.uber.org/zap. It is initialised once at startup and accessed
// globally or via context to carry per-request fields (request_id, user_id).
package logger

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ctxKey is an unexported type used as a context key to prevent collisions.
type ctxKey struct{}

var (
	global *zap.Logger
	once   sync.Once
)

// Init initialises the global singleton logger.
// In dev mode, output is colorised and human-readable.
// In prod mode, output is strict JSON for log aggregation pipelines.
// Must be called before any call to L(), With(), or WithContext().
func Init(isDev bool) {
	once.Do(func() {
		var cfg zap.Config
		if isDev {
			cfg = zap.NewDevelopmentConfig()
			cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		} else {
			cfg = zap.NewProductionConfig()
		}

		l, err := cfg.Build(zap.AddCallerSkip(0))
		if err != nil {
			panic("logger: init failed: " + err.Error())
		}
		global = l
	})
}

// L returns the global zap.Logger.
// If Init has not been called (e.g. in unit tests), a no-op logger is returned
// so callers never need to nil-check.
func L() *zap.Logger {
	if global == nil {
		return zap.NewNop()
	}
	return global
}

// With returns a child logger with the given fields pre-attached.
// Useful for adding module-level fields at startup:
//
//	log := logger.With(zap.String("module", "auth"))
func With(fields ...zap.Field) *zap.Logger {
	return L().With(fields...)
}

// Named returns a named child of the global logger.
func Named(name string) *zap.Logger {
	return L().Named(name)
}

// WithContext returns the logger stored in ctx by WithLogger,
// falling back to the global logger if none is present.
// Use this inside service/repo methods to carry request-scoped fields.
func WithContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return L()
}

// WithLogger attaches a child logger to ctx.
// Typically called in the request-logging middleware:
//
//	ctx = logger.WithLogger(ctx, logger.L().With(zap.String("request_id", rid)))
func WithLogger(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// Sync flushes any buffered log entries. Defer this at program exit:
//
//	defer logger.Sync()
func Sync() {
	if global != nil {
		_ = global.Sync()
	}
}
