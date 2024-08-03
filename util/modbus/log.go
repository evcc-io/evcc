package modbus

import (
	"sync"

	"github.com/grid-x/modbus"
	"github.com/volkszaehler/mbmd/meters"
)

type logger struct {
	mu     sync.RWMutex
	logger meters.Logger
}

func (l *logger) Logger(logger modbus.Logger) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger = logger
}

func (l *logger) Printf(format string, v ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.logger != nil {
		l.logger.Printf(format, v...)
	}
}
