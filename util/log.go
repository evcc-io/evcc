package util

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	jww "github.com/spf13/jwalterweatherman"
)

var loggers = map[string]*Logger{}

var (
	// OutThreshold is the default console log level
	OutThreshold = jww.LevelError

	// LogThreshold is the default log file level
	LogThreshold = jww.LevelWarn
)

// Logger wraps a jww notepad to avoid leaking implementation detail
type Logger struct {
	*jww.Notepad
}

// NewLogger creates a logger with the given log area and adds it to the registry
func NewLogger(area string) *Logger {
	notepad := jww.NewNotepad(OutThreshold, LogThreshold, os.Stdout, ioutil.Discard, area, log.Ldate|log.Ltime)
	l := &Logger{notepad}
	loggers[area] = l
	return l
}

// Loggers invokes callback for each configured logger
func Loggers(cb func(string, *Logger)) {
	for name, logger := range loggers {
		cb(name, logger)
	}
}

// LogLevel sets log level for all loggers
func LogLevel(level string) {
	OutThreshold = LogLevelToThreshold(level)
	LogThreshold = OutThreshold

	Loggers(func(name string, logger *Logger) {
		logger.SetStdoutThreshold(OutThreshold)
	})
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
		panic("invalid log level " + level)
	}
}
