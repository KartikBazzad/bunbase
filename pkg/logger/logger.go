package logger

import (
	"context"
	"log/slog"
	"os"
	"sync"
)

var (
	once   sync.Once
	logger *slog.Logger
)

// Config holds logger configuration
type Config struct {
	Level    string // DEBUG, INFO, WARN, ERROR
	Format   string // json, text
	AddSource bool
}

// Init initializes the global logger
func Init(cfg Config) {
	once.Do(func() {
		var level slog.Level
		switch cfg.Level {
		case "DEBUG":
			level = slog.LevelDebug
		case "WARN":
			level = slog.LevelWarn
		case "ERROR":
			level = slog.LevelError
		default:
			level = slog.LevelInfo
		}

		opts := &slog.HandlerOptions{
			Level:     level,
			AddSource: cfg.AddSource,
		}

		var handler slog.Handler
		if cfg.Format == "json" {
			handler = slog.NewJSONHandler(os.Stdout, opts)
		} else {
			handler = slog.NewTextHandler(os.Stdout, opts)
		}

		logger = slog.New(handler)
		slog.SetDefault(logger)
	})
}

// Get returns the global logger
func Get() *slog.Logger {
	if logger == nil {
		// Default fallback if not initialized
		Init(Config{Level: "INFO", Format: "json"})
	}
	return logger
}

// WithTraceID adds trace_id to the logger context
func WithTraceID(ctx context.Context, logger *slog.Logger) *slog.Logger {
	traceID, ok := ctx.Value("trace_id").(string)
	if !ok || traceID == "" {
		return logger
	}
	return logger.With("trace_id", traceID)
}

// Helper functions for quick logging
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}
