package log

import (
	golog "log"
)

type logWriter struct {
	l func(fmt string, args ...interface{})
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	l.l(string(p))
	return len(p), nil
}

func StdlogAdapter(l func(fmt string, args ...interface{})) *golog.Logger {
	w := &logWriter{l}
	return golog.New(w, "", golog.Ldate|golog.Ltime)
}
