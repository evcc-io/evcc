package modbus

import "github.com/evcc-io/evcc/util"

type logger struct {
	log *util.Logger
}

func (l *logger) Info(msg string) {
	l.log.DEBUG.Println(msg)
}

func (l *logger) Infof(format string, msg ...any) {
	l.log.DEBUG.Printf(format, msg...)
}

func (l *logger) Warning(msg string) {
	l.log.DEBUG.Println(msg)
}

func (l *logger) Warningf(format string, msg ...any) {
	l.log.DEBUG.Printf(format, msg...)
}

func (l *logger) Error(msg string) {
	l.log.ERROR.Println(msg)
}

func (l *logger) Errorf(format string, msg ...any) {
	l.log.ERROR.Printf(format, msg...)
}

func (l *logger) Fatal(msg string) {
	l.log.ERROR.Println(msg)
}

func (l *logger) Fatalf(format string, msg ...any) {
	l.log.ERROR.Printf(format, msg...)
}
