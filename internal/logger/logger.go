package logger

import (
	"log/slog"
	"os"
)

var log *slog.Logger

func init() {
	// initialize with a default logger (warn level, text handler)
	// this ensures log is never nil, even if Init() is not called
	log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))
}

func Init(verbose bool) {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelDebug
	}

	// use JSON handler for production, text for development
	var handler slog.Handler
	if os.Getenv("OX_ENV") == "production" {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
	}

	log = slog.New(handler)
	slog.SetDefault(log)
}

func Info(msg string, args ...any) {
	log.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	log.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	log.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	log.Error(msg, args...)
}
