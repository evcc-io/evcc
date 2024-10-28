package charger

import (
	"fmt"
	"time"
)

type ocppLogger struct{}

func print(s string) {
	fmt.Println(time.Now().Format(time.DateTime), s)
}

func (l *ocppLogger) Debug(args ...interface{}) { print(fmt.Sprint(args...)) }
func (l *ocppLogger) Debugf(format string, args ...interface{}) {
	print(fmt.Sprintf(format, args...))
}
func (l *ocppLogger) Info(args ...interface{}) { print(fmt.Sprint(args...)) }
func (l *ocppLogger) Infof(format string, args ...interface{}) {
	print(fmt.Sprintf(format, args...))
}
func (l *ocppLogger) Error(args ...interface{}) { print(fmt.Sprint(args...)) }
func (l *ocppLogger) Errorf(format string, args ...interface{}) {
	print(fmt.Sprintf(format, args...))
}
