package ocpp

import (
	"fmt"
	"strings"
)

func (cs *CS) print(s string) {
	var ok bool
	if s, ok = strings.CutPrefix(s, "sent JSON message to"); ok {
		s = "send" + s
	} else if s, ok = strings.CutPrefix(s, "received JSON message from"); ok {
		s = "recv" + s
	}
	if ok {
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
