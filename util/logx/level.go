package logx

import (
	kit "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func Error(log Logger, keyvals ...interface{}) {
	_ = level.Error(log).Log(keyvals...)
}

func Warn(log Logger, keyvals ...interface{}) {
	_ = level.Warn(log).Log(keyvals...)
}

func Info(log Logger, keyvals ...interface{}) {
	_ = level.Info(log).Log(keyvals...)
}

func Debug(log Logger, keyvals ...interface{}) {
	_ = level.Debug(log).Log(keyvals...)
}

func Trace(log Logger, keyvals ...interface{}) {
	_ = TraceLevel(log).Log(keyvals...)
}

// TraceLevel adds the non-standard level "trace"
func TraceLevel(log Logger) Logger {
	return kit.With(log, "level", "trace")
}
