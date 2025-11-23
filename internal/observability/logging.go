package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// ContextHandler wraps a slog.Handler to add trace_id from context
type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Try to get trace_id from context (set by middleware)
	if id, ok := ctx.Value("trace_id").(string); ok {
		r.AddAttrs(slog.String("trace_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func InitLogger() {
	level := slog.LevelInfo
	if str := os.Getenv("LOG_LEVEL"); str != "" {
		switch strings.ToUpper(str) {
		case "DEBUG":
			level = slog.LevelDebug
		case "WARN":
			level = slog.LevelWarn
		case "ERROR":
			level = slog.LevelError
		}
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Rename time to timestamp for consistency
			if a.Key == slog.TimeKey {
				a.Key = "timestamp"
			}
			return a
		},
	}

	var handler slog.Handler
	// Use TextHandler for local dev (readable), JSON for everything else (parsing)
	if os.Getenv("GIN_MODE") != "release" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	// Wrap with context handler
	ctxHandler := &ContextHandler{Handler: handler}
	slog.SetDefault(slog.New(ctxHandler))
}
