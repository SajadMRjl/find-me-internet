package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Setup initializes the global logger based on config
func Setup(level string) {
	var logLevel slog.Level

	switch strings.ToUpper(level) {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Create a structured text handler (easier to read than JSON in terminal)
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	
	logger := slog.New(handler)
	slog.SetDefault(logger)
}