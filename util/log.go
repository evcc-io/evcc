package util

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
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

var uiChan chan<- Param

type uiWriter struct {
	re    *regexp.Regexp
	level string
}

func (w *uiWriter) Write(p []byte) (n int, err error) {
	// trim level and timestamp
	s := string(w.re.ReplaceAll(p, []byte{}))

	uiChan <- Param{
		LoadPoint: "",
		Key:       w.level,
		Val:       strings.Trim(strconv.Quote(s), "\""),
	}

	return 0, nil
}

// CaptureLogs appends uiWriter to relevant log levels
func CaptureLogs(c chan<- Param) {
	uiChan = c

	for _, l := range loggers {
		captureLogger("warn", l.Notepad.WARN)
		captureLogger("error", l.Notepad.ERROR)
		captureLogger("error", l.Notepad.FATAL)
	}
}

func captureLogger(level string, l *log.Logger) {
	re, err := regexp.Compile(`^\[\w+\] \w+ .{19} `)
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
