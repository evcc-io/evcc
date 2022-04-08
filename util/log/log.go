package log

import (
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/util/log"
	jww "github.com/spf13/jwalterweatherman"
)

var (
	loggers = map[string]*logger{}
	levels  = map[string]jww.Threshold{}

	loggersMux sync.Mutex

	// OutThreshold is the default console log level
	OutThreshold = jww.LevelError

	// LogThreshold is the default log file level
	LogThreshold = jww.LevelWarn
)

// LogAreaPadding of log areas
var LogAreaPadding = 6

type Logger interface {
	Trace(fmt string, args ...interface{})
	Debug(fmt string, args ...interface{})
	Info(fmt string, args ...interface{})
	Error(fmt string, args ...interface{})
}

var _ Logger = (*logger)(nil)

// logger wraps a jww notepad to avoid leaking implementation detail
type logger struct {
	np *jww.Notepad
	*Redactor
}

// NewLogger creates a logger with the given log area and adds it to the registry
func NewLogger(area string) *logger {
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

	logger := &logger{
		np:       notepad,
		Redactor: redactor,
	}

	loggers[area] = logger

	return logger
}

func (l *logger) Trace(fmt string, args ...interface{}) {
	l.np.TRACE.Printf(fmt, args...)
}
func (l *logger) Debug(fmt string, args ...interface{}) {
	l.np.DEBUG.Printf(fmt, args...)
}
func (l *logger) Info(fmt string, args ...interface{}) {
	l.np.INFO.Printf(fmt, args...)
}
func (l *logger) Error(fmt string, args ...interface{}) {
	l.np.ERROR.Printf(fmt, args...)
}

// Redact adds items for redaction
func (l *logger) Redact(items ...string) *logger {
	l.Redactor.Redact(items...)
	return l
}

// Loggers invokes callback for each configured logger
func Loggers(cb func(string, *logger)) {
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

	Loggers(func(name string, logger *logger) {
		logger.np.SetStdoutThreshold(LogLevelForArea(name))
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

// CaptureLogs appends uiWriter to relevant log levels
func CaptureLogs(c chan<- Param) {
	uiChan = c

	for _, l := range loggers {
		captureLogger("warn", l.np.WARN)
		captureLogger("error", l.np.ERROR)
		captureLogger("error", l.np.FATAL)
	}
}

func captureLogger(level string, l *log.Logger) {
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
