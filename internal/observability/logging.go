package observability

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
		output = os.Stdout
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		output = &lumberjack.Logger{
			Filename:   "logs/vitalcache.log",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		}
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	slog.SetDefault(slog.New(handler))
}
