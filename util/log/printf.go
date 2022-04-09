package log

// PrintfLogger adapter interface
type PrintfLogger interface {
	// Printf print line to log
	Printf(format string, v ...interface{})
}

type printfLogger struct {
	l Logger
}

func (tl *printfLogger) Printf(fmt string, v ...interface{}) {
	tl.l.Trace(fmt, v...)
}

func PrintfAdapter(l Logger) PrintfLogger {
	return &printfLogger{l}
}
