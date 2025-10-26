package marstekvenusapi

import (
	"io"
	"net"
	"strings"

	"github.com/evcc-io/evcc/util"
)

// Sender is a marstek UDP sender
type Sender struct {
	log  *util.Logger
	addr string
	conn *net.UDPConn
}

// NewSender creates Marstek Open API UDP sender
func NewSender(log *util.Logger, addr string) (*Sender, error) {
	addr = util.DefaultPort(addr, Port)
	raddr, err := net.ResolveUDPAddr("udp", addr)

	var conn *net.UDPConn
	if err == nil {
		conn, err = net.DialUDP("udp", nil, raddr)
	}

	c := &Sender{
		log:  log,
		addr: addr,
		conn: conn,
	}

	return c, err
}

// Send msg to receiver
func (c *Sender) Send(msg string) error {
	c.log.TRACE.Printf("send to %s %v", c.addr, msg)
	_, err := io.Copy(c.conn, strings.NewReader(msg))
	return err
}
