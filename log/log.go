// Package log wraps log/slog with a single process-wide logger whose level
// is driven by QUAIL_LOG_LEVEL (debug|info|warn|error). All structured
// output goes to stderr so stdout stays clean for piped use.
package log

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	once   sync.Once
	logger *slog.Logger
)

// Logger returns the package-level slog logger, initializing it once from
// QUAIL_LOG_LEVEL.
func Logger() *slog.Logger {
	once.Do(func() {
		lvl := parseLevel(os.Getenv("QUAIL_LOG_LEVEL"))
		h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
		logger = slog.New(h)
	})
	return logger
}

// Set replaces the package-level logger. Intended for tests.
func Set(l *slog.Logger) {
	once.Do(func() {})
	logger = l
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func Debug(msg string, args ...any) { Logger().Debug(msg, args...) }
func Info(msg string, args ...any)  { Logger().Info(msg, args...) }
func Warn(msg string, args ...any)  { Logger().Warn(msg, args...) }
func Error(msg string, args ...any) { Logger().Error(msg, args...) }
