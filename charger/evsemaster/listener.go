package evsemaster

import (
	"fmt"
	"net"
	"sync"

	"github.com/evcc-io/evcc/util"
)

var (
	listenerMu     sync.Mutex
	sharedListener *Listener
)

// Listener is a singleton UDP listener that routes incoming EVSE Master packets
// to subscribers by device serial number.
//
// EVSE Master stations broadcast on port 28376 and always reply to the sender
// on the same port, so a shared listener is required – the same pattern as the
// KEBA UDP listener.
type Listener struct {
	mu      sync.RWMutex
	log     *util.Logger
	conn    *net.UDPConn
	clients map[string]chan<- *ReceivedPacket // keyed by 16-char hex serial
}

// Instance returns the singleton listener, creating it on first call.
func Instance(log *util.Logger) (*Listener, error) {
	listenerMu.Lock()
	defer listenerMu.Unlock()

	if sharedListener == nil {
		var err error
		sharedListener, err = newListener(log)
		if err != nil {
			return nil, err
		}
	}

	return sharedListener, nil
}

func newListener(log *util.Logger) (*Listener, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", Port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("evsemaster: bind :%d: %w (is another process using this port?)", Port, err)
	}

	l := &Listener{
		log:     log,
		conn:    conn,
		clients: make(map[string]chan<- *ReceivedPacket),
	}

	go l.listen()

	return l, nil
}

// Subscribe registers ch to receive all packets from the given serial.
// Calling Subscribe again with the same serial replaces the previous channel.
func (l *Listener) Subscribe(serial string, ch chan<- *ReceivedPacket) {
	l.mu.Lock()
	l.clients[serial] = ch
	l.mu.Unlock()
}

// Unsubscribe removes the subscription for the given serial.
func (l *Listener) Unsubscribe(serial string) {
	l.mu.Lock()
	delete(l.clients, serial)
	l.mu.Unlock()
}

// Send sends buf to the given address using the shared listener socket.
// This is required because the EVSE sends FROM its own ephemeral port (e.g. 11938)
// TO our port 28376, and replies must go back to that source address.
func (l *Listener) Send(buf []byte, addr *net.UDPAddr) error {
	_, err := l.conn.WriteTo(buf, addr)
	return err
}

func (l *Listener) listen() {
	buf := make([]byte, 1024)
	for {
		n, src, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			l.log.ERROR.Printf("evsemaster listener: %v", err)
			continue
		}

		pkt, err := Unpack(buf[:n])
		if err != nil {
			l.log.TRACE.Printf("evsemaster: unpack error: %v", err)
			continue
		}

		l.mu.RLock()
		ch, ok := l.clients[pkt.Serial]
		l.mu.RUnlock()

		if !ok {
			l.log.TRACE.Printf("evsemaster: no subscriber for serial %s (cmd 0x%04x)", pkt.Serial, pkt.Command)
			continue
		}

		rp := &ReceivedPacket{Packet: pkt, From: src}

		select {
		case ch <- rp:
		default:
			l.log.TRACE.Printf("evsemaster: recv channel full for %s", pkt.Serial)
		}
	}
}
