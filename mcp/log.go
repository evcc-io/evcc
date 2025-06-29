package mcp

import "github.com/evcc-io/evcc/util"

type logAdapter struct {
	log *util.Logger
}

func (l *logAdapter) Infof(format string, args ...interface{}) {
	l.log.INFO.Printf(format, args...)
}

func (l *logAdapter) Errorf(format string, args ...interface{}) {
	l.log.ERROR.Printf(format, args...)
}
