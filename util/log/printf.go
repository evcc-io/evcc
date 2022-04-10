package log

import "strings"

// PrintfLogger adapter interface
type PrintfLogger interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

type printfLogger struct {
	l func(fmt string, args ...interface{})
}

func (l *printfLogger) Println(v ...any) {
	l.l(strings.Repeat("%v ", len(v)), v...)
}

func (l *printfLogger) Printf(fmt string, v ...interface{}) {
	l.l(fmt, v...)
}

func PrintfAdapter(l func(fmt string, args ...interface{})) PrintfLogger {
	return &printfLogger{l}
}
