package aa55

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/evcc-io/evcc/util"
)

// AA55UDP is the GoodWe AA55-over-UDP source plugin transport.
//
// Two read modes are supported:
//
//	Register read: single register, value at offset 0 of response payload.
//	Block read:    whole register block, value extracted at a byte offset.
//	               Multiple source blocks sharing the same (host, pdu) pair
//	               share one UDP exchange per poll cycle via the response cache.
type AA55UDP struct {
	log      *util.Logger
	conn     *net.UDPConn
	pdu      []byte // 6-byte PDU body, no CRC
	offset   int    // byte offset into the response payload (0 for register reads)
	decode   string // int32be | uint32be | uint32nan | int16be | uint16be | float32be
	scale    float64
	useCache bool // true for block-read/cached mode, false for simple register read
}

// readConfig holds the resolved read mode configuration.
type readConfig struct {
	pdu      []byte
	offset   int
	useCache bool
}

// buildReadConfig normalizes config input and returns the resolved read mode.
func buildReadConfig(id int, pdu string, register uint16, count uint16, offset int) (readConfig, error) {
	// PDU/block mode
	if pdu != "" {
		// Reject mixed configuration where PDU and register parameters are both set.
		if register != 0 || count != 2 || id != int(InverterAddr) {
			return readConfig{}, errors.New("pdu cannot be combined with register/count/id settings")
		}
		if offset < 0 {
			return readConfig{}, fmt.Errorf("offset must be non-negative, got %d", offset)
		}
		pduBytes, err := parsePDU(pdu)
		if err != nil {
			return readConfig{}, err
		}
		return readConfig{pdu: pduBytes, offset: offset, useCache: true}, nil
	}

	// Register mode
	if count == 0 {
		return readConfig{}, errors.New("count must be ≥ 1")
	}
	if id < 0 || id > 255 {
		return readConfig{}, fmt.Errorf("id must be 0-255, got %d", id)
	}
	return readConfig{pdu: buildPDU(byte(id), register, count), useCache: false}, nil
}

// New constructs an AA55UDP from a high-level configuration. It validates
// decode, resolves the read mode (register vs PDU/block), and wraps the conn.
// The caller is responsible for dialling conn.
func New(log *util.Logger, conn *net.UDPConn, id int, pdu string, register, count uint16, offset int, decode string, scale float64) (*AA55UDP, error) {
	if err := validateDecode(decode); err != nil {
		return nil, err
	}
	cfg, err := buildReadConfig(id, pdu, register, count, offset)
	if err != nil {
		return nil, err
	}
	return &AA55UDP{
		log:      log,
		conn:     conn,
		decode:   decode,
		scale:    scale,
		pdu:      cfg.pdu,
		offset:   cfg.offset,
		useCache: cfg.useCache,
	}, nil
}

// FloatGetter implements the evcc plugin.FloatGetter interface.
func (p *AA55UDP) FloatGetter() (func() (float64, error), error) {
	return p.query, nil
}

// query fetches the payload and returns the decoded, scaled value at p.offset.
func (p *AA55UDP) query() (float64, error) {
	payload, err := p.fetch()
	if err != nil {
		return 0, err
	}

	minLen := p.offset + decodeSize(p.decode)
	if len(payload) < minLen {
		return 0, fmt.Errorf("payload too short (len=%d, need=%d)", len(payload), minLen)
	}

	v, err := decodeAt(payload, p.offset, p.decode)
	if err != nil {
		return 0, err
	}

	return v * p.scale, nil
}

// fetch returns the response payload, using caching for block-read mode.
func (p *AA55UDP) fetch() ([]byte, error) {
	packet := append(p.pdu, modbusCRC16(p.pdu)...)

	// No caching: direct send/recv
	if !p.useCache {
		raw, err := p.sendRecv(packet)
		if err != nil {
			return nil, err
		}
		payload, err := stripHeader(raw)
		if err != nil {
			return nil, err
		}
		return payload, nil
	}

	// Caching: shared reads by (addr, pdu)
	key := p.conn.RemoteAddr().String() + "/" + hex.EncodeToString(p.pdu)

	if payload, ok := cache.get(key); ok {
		p.log.TRACE.Printf("cache hit for %s pdu=%s", p.conn.RemoteAddr(), hex.EncodeToString(p.pdu))
		return payload, nil
	}

	raw, err := p.sendRecv(packet)
	if err != nil {
		return nil, err
	}

	payload, err := stripHeader(raw)
	if err != nil {
		return nil, err
	}

	cache.put(key, payload)
	return payload, nil
}

// sendRecv sends packet over p.conn and returns the raw response bytes.
func (p *AA55UDP) sendRecv(packet []byte) ([]byte, error) {
	p.log.TRACE.Printf("send to %s: %s", p.conn.RemoteAddr(), hex.EncodeToString(packet))

	if _, err := p.conn.Write(packet); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	if err := p.conn.SetReadDeadline(time.Now().Add(4 * time.Second)); err != nil {
		return nil, fmt.Errorf("deadline: %w", err)
	}

	buf := make([]byte, 512)
	n, err := p.conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	p.log.TRACE.Printf("recv from %s: %s", p.conn.RemoteAddr(), hex.EncodeToString(buf[:n]))

	return buf[:n], nil
}
