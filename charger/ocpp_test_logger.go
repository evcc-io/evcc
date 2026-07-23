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
