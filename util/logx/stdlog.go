package logx

import (
	stdlog "log"
)

type stdlogAdapter struct {
	Logger
}

func NewStdLogAdapter(log Logger) *stdlog.Logger {
	return stdlog.New(&stdlogAdapter{Logger: log}, "", stdlog.LstdFlags)
}

func (a *stdlogAdapter) Write(p []byte) (int, error) {
	err := a.Logger.Log("msg", string(p))
	return len(p), err
}
