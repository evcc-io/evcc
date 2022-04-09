package log

import (
	"io"
	golog "log"

	jww "github.com/spf13/jwalterweatherman"
)

type Logger interface {
	Trace(fmt string, args ...interface{})
	Debug(fmt string, args ...interface{})
	Info(fmt string, args ...interface{})
	Warn(fmt string, args ...interface{})
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
	notepad := jww.NewNotepad(level, level, redactor, io.Discard, padded, golog.Ldate|golog.Ltime)

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
func (l *logger) Warn(fmt string, args ...interface{}) {
	l.np.WARN.Printf(fmt, args...)
}
func (l *logger) Error(fmt string, args ...interface{}) {
	l.np.ERROR.Printf(fmt, args...)
}

// Redact adds items for redaction
func (l *logger) Redact(items ...string) *logger {
	l.Redactor.Redact(items...)
	return l
}
