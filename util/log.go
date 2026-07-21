package util

import (
	"fmt"
	"log"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/util/logstash"
)

// Log levels, extending slog by Trace and Fatal
const (
	LevelTrace = logstash.LevelTrace
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
	LevelFatal = logstash.LevelFatal
)

var (
	loggers = map[string]*Logger{}
	levels  = map[string]slog.Level{}

	loggersMux sync.Mutex

	// OutThreshold is the default console log level
	OutThreshold = slog.LevelInfo
)

// LogAreaPadding of log areas
var LogAreaPadding = 6

// Attribute keys for structured log entries
const (
	ComponentKey = "component" // core component: site, loadpoint
	DeviceKey    = "device"    // device class: meter, charger, vehicle, tariff, circuit, messenger
	TitleKey     = "title"     // user-facing title
	IdKey        = "id"        // loadpoint id
	TransportKey = "transport" // trace transport: http, mqtt, modbus
)

// Logger provides slog-based, leveled logging per log area. The printf-style
// level loggers remain for legacy call sites; new code should prefer slog.
type Logger struct {
	*slog.Logger
	*Redactor

	TRACE, DEBUG, INFO, WARN, ERROR, FATAL *log.Logger

	handler *handler
}

// NewLogger creates a logger with the given log area and adds it to the registry
func NewLogger(area string) *Logger {
	return newLogger(area, 0)
}

// NewLoggerWithLoadpoint creates a logger with reference to at loadpoint
func NewLoggerWithLoadpoint(area string, lp int) *Logger {
	return newLogger(area, lp).With(ComponentKey, "loadpoint", IdKey, lp)
}

func newLogger(area string, lp int) *Logger {
	loggersMux.Lock()
	defer loggersMux.Unlock()

	if logger, ok := loggers[area]; ok {
		return logger
	}

	level := new(slog.LevelVar)
	level.Set(logLevelForArea(area))

	h := &handler{
		area:     area,
		padded:   fmt.Sprintf("%-*s", LogAreaPadding, area),
		level:    level,
		redactor: new(Redactor),
		lp:       lp,
	}

	logger := newHandlerLogger(h)
	loggers[area] = logger

	return logger
}

func newHandlerLogger(h *handler) *Logger {
	return &Logger{
		Logger:   slog.New(h),
		Redactor: h.redactor,
		TRACE:    slog.NewLogLogger(h, LevelTrace),
		DEBUG:    slog.NewLogLogger(h, LevelDebug),
		INFO:     slog.NewLogLogger(h, LevelInfo),
		WARN:     slog.NewLogLogger(h, LevelWarn),
		ERROR:    slog.NewLogLogger(h, LevelError),
		FATAL:    slog.NewLogLogger(h, LevelFatal),
		handler:  h,
	}
}

// With returns a derived logger carrying additional attributes. It shares
// area, log level and redaction with its parent.
func (l *Logger) With(args ...any) *Logger {
	return newHandlerLogger(l.handler.with(args))
}

// Redact adds items for redaction
func (l *Logger) Redact(items ...string) *Logger {
	l.Redactor.Redact(items...)
	return l
}

// Loggers invokes callback for each configured logger
func Loggers(cb func(string, *Logger)) {
	for name, logger := range loggers {
		cb(name, logger)
	}
}

// logLevelForArea gets the log level for given log area
func logLevelForArea(area string) slog.Level {
	level, ok := levels[strings.ToLower(area)]
	if !ok {
		level = OutThreshold
	}
	return level
}

// LogLevel sets log level for all loggers
func LogLevel(defaultLevel string, areaLevels map[string]string) {
	// default level
	OutThreshold = logstash.LogLevelToThreshold(defaultLevel)

	// area levels
	for area, level := range areaLevels {
		area = strings.ToLower(area)
		levels[area] = logstash.LogLevelToThreshold(level)
	}

	Loggers(func(name string, logger *Logger) {
		logger.handler.level.Set(logLevelForArea(name))
	})
}

var (
	uiChanMux sync.RWMutex
	uiChan    chan<- Param
)

// CaptureLogs routes warnings and errors to the ui
func CaptureLogs(c chan<- Param) {
	uiChanMux.Lock()
	defer uiChanMux.Unlock()

	if uiChan == nil {
		uiChan = c
	}
}

func uiCapture(level string, lp int, msg string) {
	uiChanMux.RLock()
	defer uiChanMux.RUnlock()

	if uiChan == nil {
		return
	}

	val := struct {
		Message   string `json:"message"`
		Level     string `json:"level"`
		Loadpoint int    `json:"lp,omitempty"`
	}{
		Message:   strings.Trim(strconv.Quote(strings.TrimSpace(msg)), "\""),
		Level:     level,
		Loadpoint: lp,
	}

	uiChan <- Param{Key: "log", Val: val}
}
