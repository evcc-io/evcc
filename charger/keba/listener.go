package keba

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/andig/evcc/util"
)

const (
	udpBufferSize = 1024

	// Port is the KEBA UDP port
	Port = 7090

	// OK is the KEBA confirmation message
	OK = "TCH-OK :done"

	// Any subscriber receives all messages
	Any = "<any>"
)

// Instance is the KEBA listener instance
// This is needed since KEBAs ignore the sender port and always UDP back to port 7090
var Instance *Listener

// UDPMsg transports the KEBA response. Report is any of Report1,2,3
type UDPMsg struct {
	Addr    string
	Message []byte
	Report  *Report
}

// Listener singleton listens for KEBA UDP messages
type Listener struct {
	mux     sync.Mutex
	log     *util.Logger
	conn    *net.UDPConn
	clients map[string]chan<- UDPMsg
}

// New creates a UDP listener that clients can subscribe to
func New(log *util.Logger) (*Listener, error) {
	laddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", Port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, err
	}

	l := &Listener{
		log:     log,
		conn:    conn,
		clients: make(map[string]chan<- UDPMsg),
	}

	go l.listen()

	return l, nil
}

// Subscribe adds a client address and message channel
func (l *Listener) Subscribe(addr string, c chan<- UDPMsg) error {
	l.mux.Lock()
	defer l.mux.Unlock()

	if _, exists := l.clients[addr]; exists {
		return fmt.Errorf("duplicate subscription: %s", addr)
	}

	l.clients[addr] = c
	return nil
}

func (l *Listener) listen() {
	b := make([]byte, udpBufferSize)

	for {
		read, addr, err := l.conn.ReadFrom(b)
		if err != nil {
			l.log.ERROR.Printf("listener: %v", err)
			continue
		}

		body := strings.TrimSpace(string(b[:read]))
		l.log.TRACE.Printf("recv from %s %v", addr.String(), body)

		msg := UDPMsg{
			Addr:    addr.String(),
			Message: []byte(body),
		}

		if body != OK {
			var report Report
			if err := json.Unmarshal([]byte(body), &report); err != nil {
				l.log.WARN.Printf("listener: %v", err)
				continue
			}

			msg.Report = &report
		}

		l.send(msg)
	}
}

// addrMatches checks if either message sender or serial matched given addr
func (l *Listener) addrMatches(addr string, msg UDPMsg) bool {
	switch {
	case addr == Any:
		return true
	case addr == msg.Addr:
		return true
	case msg.Report != nil && addr == msg.Report.Serial:
		return true
	default:
		return false
	}
}

func (l *Listener) send(msg UDPMsg) {
	l.mux.Lock()
	defer l.mux.Unlock()

	for addr, client := range l.clients {
		if l.addrMatches(addr, msg) {
			select {
			case client <- msg:
			default:
				l.log.TRACE.Println("recv: listener blocked")
			}
			break
		}
	}
}
