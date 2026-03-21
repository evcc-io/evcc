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
	c.lst.Subscribe(c.serial, ch)
}

// Unsubscribe removes the subscription for this device's serial.
func (c *Connection) Unsubscribe() {
	c.lst.Unsubscribe(c.serial)
}
