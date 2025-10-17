package logging

import (
	"log/slog"
	"os"
)

func InitLogger() {
	var handler slog.Handler

	// Use a more readable logger for development
	if os.Getenv("GIN_MODE") != "release" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		// Use JSON logger for production for easier parsing by log collectors
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
