package charger

import (
	"fmt"
	"time"
)

type ocppLogger struct{}

func (l *ocppLogger) print(s string) {
	fmt.Println(time.Now().Format(time.DateTime), s)
}

func (l *ocppLogger) Debug(args ...interface{}) { l.print(fmt.Sprint(args...)) }
func (l *ocppLogger) Debugf(format string, args ...interface{}) {
	l.print(fmt.Sprintf(format, args...))
}
func (l *ocppLogger) Info(args ...interface{}) { l.print(fmt.Sprint(args...)) }
func (l *ocppLogger) Infof(format string, args ...interface{}) {
	l.print(fmt.Sprintf(format, args...))
}
func (l *ocppLogger) Error(args ...interface{}) { l.print(fmt.Sprint(args...)) }
func (l *ocppLogger) Errorf(format string, args ...interface{}) {
	l.print(fmt.Sprintf(format, args...))
}
