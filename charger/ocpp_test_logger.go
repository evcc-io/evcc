package charger

import (
	"fmt"
	"testing"
	"time"
)

type ocppLogger struct {
	t *testing.T
}

func print(t *testing.T, s string) {
	t.Log((time.Now().Format(time.DateTime)), s)
}

func (l *ocppLogger) Debug(args ...interface{}) { print(l.t, fmt.Sprint(args...)) }
func (l *ocppLogger) Debugf(format string, args ...interface{}) {
	print(l.t, fmt.Sprintf(format, args...))
}
func (l *ocppLogger) Info(args ...interface{}) { print(l.t, fmt.Sprint(args...)) }
func (l *ocppLogger) Infof(format string, args ...interface{}) {
	print(l.t, fmt.Sprintf(format, args...))
}
func (l *ocppLogger) Error(args ...interface{}) { print(l.t, fmt.Sprint(args...)) }
func (l *ocppLogger) Errorf(format string, args ...interface{}) {
	print(l.t, fmt.Sprintf(format, args...))
}
