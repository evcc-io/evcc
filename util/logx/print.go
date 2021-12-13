package logx

import (
	"fmt"
)

type Printer interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

type printAdapter struct {
	Logger
}

func NewPrintAdapter(log Logger) *printAdapter {
	return &printAdapter{Logger: log}
}

func (l *printAdapter) Printf(format string, v ...interface{}) {
	_ = l.Logger.Log("msg", fmt.Sprintf(format, v...))
}

func (l *printAdapter) Println(v ...interface{}) {
	_ = l.Logger.Log("msg", fmt.Sprintln(v...))
}
