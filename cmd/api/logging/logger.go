package logging

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger() {
	var output io.Writer
	var handler slog.Handler

	if os.Getenv("GIN_MODE") != "release" {
		// In development, log to the console with colors
		output = os.Stdout
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		// In production, log to a file with rotation
		output = &lumberjack.Logger{
			Filename:   "logs/vitalcache.log",
			MaxSize:    10, // megabytes
			MaxBackups: 3,
			MaxAge:     28, // days
			Compress:   true,
		}
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
