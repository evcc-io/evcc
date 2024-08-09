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

func (l *logger) WithLogger(logger modbus.Logger, fun func() ([]byte, error)) ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger = logger
	return fun()
}

// Printf implements modbus.Logger interface.
// Must always be called while being wrapped in WithLogger, hence the lock is held.
func (l *logger) Printf(format string, v ...interface{}) {
	if l.logger != nil {
		l.logger.Printf(format, v...)
	}
}
