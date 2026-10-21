package logstash

import (
	"slices"
	"strings"

	jww "github.com/spf13/jwalterweatherman"
)

// Levels lists all valid log level names, ordered from most to least severe
var Levels = []string{"fatal", "error", "warn", "info", "debug", "trace"}

// ValidLevel reports whether level is a recognized log level name
func ValidLevel(level string) bool {
	return slices.Contains(Levels, strings.ToLower(level))
}

// LogLevelToThreshold converts log level string to a jww Threshold
func LogLevelToThreshold(level string) jww.Threshold {
	switch strings.ToUpper(level) {
	case "FATAL":
		return jww.LevelFatal
	case "ERROR":
		return jww.LevelError
	case "WARN":
		return jww.LevelWarn
	case "INFO":
		return jww.LevelInfo
	case "DEBUG":
		return jww.LevelDebug
	case "TRACE":
		return jww.LevelTrace
	default:
		return jww.LevelError
	}
}
