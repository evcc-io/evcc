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
	// var ok bool
	// if s, ok = strings.CutPrefix(s, "sent JSON message to"); ok {
	// 	s = "send" + s
	// } else if s, ok = strings.CutPrefix(s, "received JSON message from"); ok {
	// 	s = "recv" + s
	// } else {
	// 	ok = true
	// }
	// if ok {
	t.Log((time.Now().Format(time.DateTime)), s)
	// }
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
