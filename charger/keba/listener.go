package keba

import (
	"encoding/json"
	"net"
	"strings"
	"sync"

	"github.com/andig/evcc/util"
)

const (
	udpBufferSize = 1024

	// OK is the KEBA confirmation message
	OK = "TCH-OK :done"
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
func New(log *util.Logger, addr string) *Listener {
	laddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	l := &Listener{
		log:  log,
		conn: conn,
	}

	go l.listen()

	return l
}

// Subscribe adds a client address and message channel
func (l *Listener) Subscribe(addr string, c chan<- UDPMsg) {
	l.mux.Lock()
	defer l.mux.Unlock()

	if l.clients == nil {
		l.clients = make(map[string]chan<- UDPMsg)
	}

	l.clients[addr] = c
}

func (l *Listener) listen() {
	b := make([]byte, udpBufferSize)

	for {
		read, addr, err := l.conn.ReadFrom(b)
		if err != nil {
			l.log.WARN.Printf("listener: %v", err)
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

func (l *Listener) send(msg UDPMsg) {
	l.mux.Lock()
	defer l.mux.Unlock()

	for addr, client := range l.clients {
		if addr == msg.Addr {
			select {
			case client <- msg:
			default:
				l.log.TRACE.Println("listener: recv blocked")
			}
			break
		}
	}
}
