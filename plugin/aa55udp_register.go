package plugin

import (
	"encoding/hex"
	"fmt"
	"time"
)

// fetchRegister performs a single UDP request/response exchange for the
// register read mode.  The response payload is returned without caching —
// each call to the getter triggers a fresh UDP exchange.
func (p *AA55UDP) fetchRegister() ([]byte, error) {
	packet := append(p.pdu, modbusCRC16(p.pdu)...)

	raw, err := p.sendRecv(packet)
	if err != nil {
		return nil, err
	}

	payload, err := stripAA55Header(raw)
	if err != nil {
		return nil, fmt.Errorf("aa55udp: %w", err)
	}

	return payload, nil
}

// sendRecv sends packet over p.conn and returns the raw response bytes.
// Used by both register read and block read modes.
func (p *AA55UDP) sendRecv(packet []byte) ([]byte, error) {
	p.log.TRACE.Printf("send to %s: %s", p.raddr, hex.EncodeToString(packet))

	if _, err := p.conn.Write(packet); err != nil {
		return nil, fmt.Errorf("aa55udp write: %w", err)
	}

	if err := p.conn.SetReadDeadline(time.Now().Add(4 * time.Second)); err != nil {
		return nil, fmt.Errorf("aa55udp deadline: %w", err)
	}

	buf := make([]byte, 512)
	n, err := p.conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("aa55udp read: %w", err)
	}

	p.log.TRACE.Printf("recv from %s: %s", p.raddr, hex.EncodeToString(buf[:n]))

	return buf[:n], nil
}
