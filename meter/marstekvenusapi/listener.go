package marstekvenusapi

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/util"
)

const (
	udpBufferSize = 2048

	// Port is the standard marstek UDP port
	Port = DEFAULT_PORT

	// Any subscriber receives all messages
	Any = "<any>"
)

// instance is the marstek listener instance
// This is needed since KEBAs ignore the sender port and always UDP back to port 7090
var (
	mu       sync.Mutex
	instance *Listener
)

// UDPMsg transports the Marstek response. Response is any of the defined Responses
type UDPMsg struct {
	Addr     string
	Message  []byte
	Response *Response
}

// Listener singleton listens for Marstek UDP messages
type Listener struct {
	mux     sync.Mutex
	log     *util.Logger
	conn    *net.UDPConn
	clients map[string]chan<- UDPMsg
	cache   map[string]string
}

func Instance(log *util.Logger) (*Listener, error) {
	mu.Lock()
	defer mu.Unlock()

	var err error
	if instance == nil {
		instance, err = New(log)
	}

	return instance, err
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
		cache:   make(map[string]string),
	}

	go l.listen()

	return l, nil
}

// Subscribe adds a client address or serial and message channel to the list of subscribers
func (l *Listener) Subscribe(addr string, c chan<- UDPMsg) {
	l.mux.Lock()
	defer l.mux.Unlock()

	l.clients[addr] = c
}

func (l *Listener) listen() {
	b := make([]byte, udpBufferSize)

	for {
		count, remoteAddr, err := l.conn.ReadFrom(b)
		if err != nil {
			l.log.TRACE.Printf("listener: %v", err)
			continue
		}

		body := strings.TrimSpace(string(b[:count]))
		l.log.TRACE.Printf("recv from %s %v", remoteAddr.String(), body)

		msg := UDPMsg{
			Addr:    remoteAddr.String(),
			Message: []byte(body),
		}

		if body != "" { //ok
			var response Response
			if err := json.Unmarshal([]byte(body), &response); err != nil {
				// ignore error during detection when sending report request to localhost
				if body != "report 1" {
					l.log.WARN.Printf("recv: invalid message: %v", err)
				}
				continue
			}

			msg.Response = &response
		}

		l.send(msg)
	}
}

// addrMatches checks if either message sender or serial matches given addr
func (l *Listener) addrMatches(addr string, msg UDPMsg) bool {
	switch {
	case addr == Any:
		return true

	case addr == msg.Addr:
		return true

	// simple response like TCH :OK where cached serial for sender address matches
	case l.cache[addr] == msg.Addr:
		return true

	// report response with matching serial
	//case msg.Response != nil && addr == msg.Response.Serial:
	// cache address for serial to make simple TCH :OK messages routable using serial
	//	l.cache[msg.Report.Serial] = msg.Addr
	//		return true

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
