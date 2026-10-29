package logstash

import (
	"log/slog"
	"strings"
)

// Log levels, extending slog by Trace and Fatal
const (
	LevelTrace = slog.Level(-8)
	LevelFatal = slog.Level(12)
)

// LogLevelToThreshold converts log level string to a slog.Level
func LogLevelToThreshold(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "FATAL":
		return LevelFatal
	case "ERROR":
		return slog.LevelError
	case "WARN":
		return slog.LevelWarn
	case "INFO":
		return slog.LevelInfo
	case "DEBUG":
		return slog.LevelDebug
	case "TRACE":
		return LevelTrace
	default:
		return slog.LevelError
	}
}
