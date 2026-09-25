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

// traceRecv logs a raw frame received from a charger, matching the format the
// ocpp-go library uses for frames it handles itself. Used for forwarder frames
// that bypass the library handler and would otherwise go untraced.
func (cs *CS) traceRecv(id string, data []byte) {
	cs.log.TRACE.Printf("recv %s: %s", id, data)
}

// traceSend logs a raw frame sent to a charger. origin marks its source:
// "" for evcc-generated frames, "upstream" for frames proxied from the
// upstream OCPP server.
func (cs *CS) traceSend(id, origin string, data []byte) {
	if origin != "" {
		cs.log.TRACE.Printf("send %s (%s): %s", id, origin, data)
		return
	}
	cs.log.TRACE.Printf("send %s: %s", id, data)
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
