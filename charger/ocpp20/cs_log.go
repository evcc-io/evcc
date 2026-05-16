package ocpp20

import (
	"fmt"
	"strings"
)

func (cs *CSMS) print(s string) {
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

func (cs *CSMS) Debug(args ...any) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CSMS) Debugf(f string, args ...any) {
	cs.print(fmt.Sprintf(f, args...))
}

func (cs *CSMS) Info(args ...any) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CSMS) Infof(f string, args ...any) {
	cs.print(fmt.Sprintf(f, args...))
}

func (cs *CSMS) Error(args ...any) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CSMS) Errorf(f string, args ...any) {
	cs.print(fmt.Sprintf(f, args...))
}
