package ocpp

import (
	"fmt"
	"strings"
)

func (cs *CS) print(s string) {
	if strings.Contains(s, "JSON message") {
		cs.log.TRACE.Println(s)
	}
}

func (cs *CS) Debug(args ...interface{}) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CS) Debugf(f string, args ...interface{}) {
	cs.print(fmt.Sprintf(f, args...))
}

func (cs *CS) Info(args ...interface{}) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CS) Infof(f string, args ...interface{}) {
	cs.print(fmt.Sprintf(f, args...))
}

func (cs *CS) Error(args ...interface{}) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CS) Errorf(f string, args ...interface{}) {
	cs.print(fmt.Sprintf(f, args...))
}
