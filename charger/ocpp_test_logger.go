package charger

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type ocppLogger struct {
	mu sync.Mutex
	t  *testing.T
}

// open and close bind the logger to the currently running test. The logger is
// registered with ocppj once for the whole binary: rebinding it per suite run
// would race with charge point goroutines that outlive the run.
func (l *ocppLogger) open(t *testing.T) {
	l.mu.Lock()
	l.t = t
	l.mu.Unlock()
}

func (l *ocppLogger) close() {
	l.mu.Lock()
	l.t = nil
	l.mu.Unlock()
}

func (l *ocppLogger) print(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.t != nil {
		l.t.Log(time.Now().Format(time.DateTime), s)
	}
}

func (l *ocppLogger) Debug(args ...any)                 { l.print(fmt.Sprint(args...)) }
func (l *ocppLogger) Debugf(format string, args ...any) { l.print(fmt.Sprintf(format, args...)) }
func (l *ocppLogger) Info(args ...any)                  { l.print(fmt.Sprint(args...)) }
func (l *ocppLogger) Infof(format string, args ...any)  { l.print(fmt.Sprintf(format, args...)) }
func (l *ocppLogger) Error(args ...any)                 { l.print(fmt.Sprint(args...)) }
func (l *ocppLogger) Errorf(format string, args ...any) { l.print(fmt.Sprintf(format, args...)) }
