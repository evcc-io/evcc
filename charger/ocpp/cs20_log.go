package ocpp

import (
	"fmt"
	"strings"
)

func (cs *CSMS20) print(s string) {
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

func (cs *CSMS20) Debug(args ...any) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CSMS20) Debugf(f string, args ...any) {
	cs.print(fmt.Sprintf(f, args...))
}

func (cs *CSMS20) Info(args ...any) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CSMS20) Infof(f string, args ...any) {
	cs.print(fmt.Sprintf(f, args...))
}

func (cs *CSMS20) Error(args ...any) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CSMS20) Errorf(f string, args ...any) {
	cs.print(fmt.Sprintf(f, args...))
}
