package utils

import (
	"fmt"
	"log/slog"
	"strings"
)

func ParseLogLevel(logLevel string) (slog.Level, error) {
	switch strings.ToLower(logLevel) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	case "fatal":
		return slog.LevelError, nil // No fatal in slog, map to error.
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level: %s", logLevel)
	}
}
