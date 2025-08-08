package mcp

import "github.com/evcc-io/evcc/util"

type logger struct {
	log *util.Logger
}

func (l *logger) Infof(format string, v ...any) {
	l.log.TRACE.Printf(format, v...)
}

func (l *logger) Errorf(format string, v ...any) {
	l.log.WARN.Printf(format, v...)
}
