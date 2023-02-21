package util

import (
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"

	jww "github.com/spf13/jwalterweatherman"
)

var (
	loggers = map[string]*Logger{}
	levels  = map[string]jww.Threshold{}

	loggersMux sync.Mutex

	// OutThreshold is the default console log level
	OutThreshold = jww.LevelError

	// LogThreshold is the default log file level
	LogThreshold = jww.LevelWarn
)

// LogAreaPadding of log areas
var LogAreaPadding = 6

// Logger wraps a jww notepad to avoid leaking implementation detail
type Logger struct {
	*jww.Notepad
	*Redactor
}

// NewLogger creates a logger with the given log area and adds it to the registry
func NewLogger(area string) *Logger {
	loggersMux.Lock()
	defer loggersMux.Unlock()

	if logger, ok := loggers[area]; ok {
		return logger
	}

	padded := area
	for len(padded) < LogAreaPadding {
		padded = padded + " "
	}

	level := LogLevelForArea(area)
	redactor := new(Redactor)
	notepad := jww.NewNotepad(level, level, redactor, io.Discard, padded, log.Ldate|log.Ltime)

	logger := &Logger{
		Notepad:  notepad,
		Redactor: redactor,
	}

	// capture loggers created after uiChan is initialized
	if uiChan != nil {
		captureLogger(logger)
	}

	loggers[area] = logger

	return logger
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

// LogLevelForArea gets the log level for given log area
func LogLevelForArea(area string) jww.Threshold {
	level, ok := levels[strings.ToLower(area)]
	if !ok {
		level = OutThreshold
	}
	return level
}

// LogLevel sets log level for all loggers
func LogLevel(defaultLevel string, areaLevels map[string]string) {
	// default level
	OutThreshold = LogLevelToThreshold(defaultLevel)
	LogThreshold = OutThreshold

	// area levels
	for area, level := range areaLevels {
		area = strings.ToLower(area)
		levels[area] = LogLevelToThreshold(level)
	}

	Loggers(func(name string, logger *Logger) {
		logger.SetStdoutThreshold(LogLevelForArea(name))
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

var uiChan chan<- Param

type uiWriter struct {
	re    *regexp.Regexp
	level string
}

func (w *uiWriter) Write(p []byte) (n int, err error) {
	// trim level and timestamp
	s := string(w.re.ReplaceAll(p, []byte{}))

	uiChan <- Param{
		Key: w.level,
		Val: strings.Trim(strconv.Quote(strings.TrimSpace(s)), "\""),
	}

	return 0, nil
}

// CaptureLogs appends uiWriter to relevant log levels for
// loggers created before uiChan is initialized
func CaptureLogs(valueChan chan<- Param) {
	if uiChan != nil {
		panic("log capturing already initialized")
	}

	uiChan = valueChan

	for _, l := range loggers {
		captureLogger(l)
	}
}

func captureLogger(l *Logger) {
	captureLogLevel("warn", l.Notepad.WARN)
	captureLogLevel("error", l.Notepad.ERROR)
	captureLogLevel("error", l.Notepad.FATAL)
}

func captureLogLevel(level string, l *log.Logger) {
	re, err := regexp.Compile(`^\[[a-zA-Z0-9-]+\s*\] \w+ .{19} `)
	if err != nil {
		panic(err)
	}

	ui := uiWriter{
		re:    re,
		level: level,
	}

	mw := io.MultiWriter(l.Writer(), &ui)
	l.SetOutput(mw)
}
