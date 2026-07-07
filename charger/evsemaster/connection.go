package evsemaster

import (
	"net"

	"github.com/evcc-io/evcc/util"
)

// Connection holds per-device credentials and routes sends through the shared listener.
// Multiple Connection instances with different serials can coexist; the Listener
// routes incoming packets to each by serial number.
type Connection struct {
	lst      *Listener
	serial   string
	password string
	recv     chan<- *ReceivedPacket // channel passed to Subscribe, used to identify on Unsubscribe
}

// NewConnection creates a Connection for a device identified by serial and password.
func NewConnection(log *util.Logger, serial, password string) (*Connection, error) {
	lst, err := Instance(log)
	if err != nil {
		return nil, err
	}
	return &Connection{lst: lst, serial: serial, password: password}, nil
}

// Send packs a command with device credentials and sends it to the given EVSE address.
func (c *Connection) Send(cmd uint16, payload []byte, addr *net.UDPAddr) error {
	pkt := &Packet{
		Serial:   c.serial,
		Password: c.password,
		Command:  cmd,
		Payload:  payload,
	}
	buf, err := pkt.Pack()
	if err != nil {
		return err
	}
	return c.lst.Send(buf, addr)
}

// Subscribe registers ch to receive all packets from this device's serial.
func (c *Connection) Subscribe(ch chan<- *ReceivedPacket) {
	c.recv = ch
	c.lst.Subscribe(c.serial, ch)
}

// Reclaim registers ch only if no subscriber currently holds the slot.
// Used on keepalive ticks so the long-running instance does not displace
// a temporary validate instance that is still active.
func (c *Connection) Reclaim(ch chan<- *ReceivedPacket) {
	c.recv = ch
	c.lst.Reclaim(c.serial, ch)
}

// Unsubscribe removes this connection's subscription only if its channel is
// still the active one, so a stale unsubscribe cannot displace a newer subscriber.
func (c *Connection) Unsubscribe() {
	c.lst.Unsubscribe(c.serial, c.recv)
}

// Addr gets or sets the last known EVSE address for this device.
func (c *Connection) Addr(addr *net.UDPAddr) *net.UDPAddr {
	return c.lst.Addr(c.serial, c.password, addr)
}
