package log

import "os"

var root = NewLogger("main")

func Trace(fmt string, args ...interface{}) {
	root.Trace(fmt, args...)
}
func Debug(fmt string, args ...interface{}) {
	root.Debug(fmt, args...)
}
func Info(fmt string, args ...interface{}) {
	root.Info(fmt, args...)
}
func Warn(fmt string, args ...interface{}) {
	root.Warn(fmt, args...)
}
func Error(fmt string, args ...interface{}) {
	root.Error(fmt, args...)
}

func Fatal(fmt string, args ...interface{}) {
	root.Error(fmt, args...)
	os.Exit(1)
}
