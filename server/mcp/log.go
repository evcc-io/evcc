package mcp

import "github.com/evcc-io/evcc/util"

// stdLogger wraps the standard library's log.Logger.
type stdLogger struct {
	logger *util.Logger
}

func (l *stdLogger) Infof(format string, v ...any) {
	l.logger.TRACE.Printf("INFO: "+format, v...)
}

func (l *stdLogger) Errorf(format string, v ...any) {
	l.logger.TRACE.Printf("ERROR: "+format, v...)
}
