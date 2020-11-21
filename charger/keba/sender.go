package keba

import (
	"io"
	"net"
	"strings"

	"github.com/andig/evcc/util"
)

// Sender is a KEBA UDP sender
type Sender struct {
	conn *net.UDPConn
}

// NewSender creates KEBA UDP sender
func NewSender(addr string) (*Sender, error) {
	addr = util.DefaultPort(addr, Port)
	raddr, err := net.ResolveUDPAddr("udp", addr)

	var conn *net.UDPConn
	if err == nil {
		conn, err = net.DialUDP("udp", nil, raddr)
	}

	c := &Sender{
		conn: conn,
	}

	return c, err
}

// Send msg to receiver
func (c *Sender) Send(msg string) error {
	_, err := io.Copy(c.conn, strings.NewReader(msg))
	return err
}
