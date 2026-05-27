package ocpp

import (
	"fmt"
	"strings"
)

func (cs *CS) print(s string) {
	// for _, p := range []string{
	// 	"completed request",
	// 	"dispatched request",
	// 	"enqueued CALL",
	// 	"handling incoming",
	// 	"sent CALL",
	// 	"started timeout",
	// 	"timeout canceled",
	// } {
	// 	if strings.HasPrefix(s, p) {
	// 		return
	// 	}
	// }

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

func (cs *CS) Debug(args ...any) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CS) Debugf(f string, args ...any) {
	cs.print(fmt.Sprintf(f, args...))
}

func (cs *CS) Info(args ...any) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CS) Infof(f string, args ...any) {
	cs.print(fmt.Sprintf(f, args...))
}

func (cs *CS) Error(args ...any) {
	cs.print(fmt.Sprintln(args...))
}

func (cs *CS) Errorf(f string, args ...any) {
	cs.print(fmt.Sprintf(f, args...))
}
