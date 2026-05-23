package plugin

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/evcc-io/evcc/plugin/aa55"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// AA55UDP is an evcc source plugin for the GoodWe AA55-over-UDP wire protocol.
//
// Two read modes are supported, selected at construction time:
//
//	Register read: single register, value at offset 0 of response payload.
//	Block read:    whole register block, value extracted at a byte offset.
//	               Multiple source blocks sharing the same (host, pdu) pair
//	               share one UDP exchange per poll cycle via a response cache.
//
// See package plugin/aa55 for the wire protocol primitives.
type AA55UDP struct {
	log      *util.Logger
	conn     *net.UDPConn
	pdu      []byte // 6-byte PDU body, no CRC
	offset   int    // byte offset into the response payload (0 for register reads)
	decode   string // int32be | uint32be | uint32nan | int16be | uint16be | float32be
	scale    float64
	useCache bool // true for block-read/cached mode, false for simple register read
}

func init() {
	registry.AddCtx("aa55udp", NewAA55UDPFromConfig)
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
		if register != 0 || count != 2 || id != int(aa55.InverterAddr) {
			return readConfig{}, errors.New("aa55udp: pdu cannot be combined with register/count/id settings")
		}
		if offset < 0 {
			return readConfig{}, fmt.Errorf("aa55udp: offset must be non-negative, got %d", offset)
		}
		pduBytes, err := aa55.ParsePDU(pdu)
		if err != nil {
			return readConfig{}, fmt.Errorf("aa55udp: %w", err)
		}
		return readConfig{pdu: pduBytes, offset: offset, useCache: true}, nil
	}

	// Register mode
	if count == 0 {
		return readConfig{}, errors.New("aa55udp: count must be ≥ 1")
	}
	if id < 0 || id > 255 {
		return readConfig{}, fmt.Errorf("aa55udp: id must be 0-255, got %d", id)
	}
	return readConfig{pdu: aa55.BuildPDU(byte(id), register, count), useCache: false}, nil
}

// NewAA55UDPFromConfig creates an AA55UDP plugin.
//
// Register read mode (single register):
//
//	source:   aa55udp
//	host:     192.168.1.26
//	id:       127          # 0x7F for DT/DNS/ES/EM (default); 247 (0xF7) for ET/EH/BT/BH
//	register: 30127
//	count:    2            # 1 = 16-bit, 2 = 32-bit
//	decode:   int32be
//	scale:    1.0
//
// Block read mode (whole block, extract value at offset):
//
//	source:   aa55udp
//	host:     192.168.1.26   # inverter IP; port 8899 is always used
//	pdu:      f703891c007d   # raw 6-byte PDU (hex)
//	offset:   54             # byte offset of the value within the block payload
//	decode:   int32be        # int32be | uint32be | uint32nan | int16be | uint16be | float32be
//	scale:    1.0            # optional multiplier (default 1.0)
func NewAA55UDPFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		Host     string  `mapstructure:"host"`
		Id       int     `mapstructure:"id"`
		PDU      string  `mapstructure:"pdu"`
		Register uint16  `mapstructure:"register"`
		Count    uint16  `mapstructure:"count"`
		Offset   int     `mapstructure:"offset"`
		Decode   string  `mapstructure:"decode"`
		Scale    float64 `mapstructure:"scale"`
	}{
		Id:    int(aa55.InverterAddr),
		Count: 2,
		Scale: 1.0,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := aa55.ValidateDecode(cc.Decode); err != nil {
		return nil, fmt.Errorf("aa55udp: %w", err)
	}

	raddr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(cc.Host, "8899"))
	if err != nil {
		return nil, fmt.Errorf("aa55udp: resolve %s: %w", cc.Host, err)
	}
	dialer := &net.Dialer{Timeout: request.Timeout}
	conn, err := dialer.DialUDP(ctx, "udp4", netip.AddrPort{}, raddr.AddrPort())
	if err != nil {
		return nil, fmt.Errorf("aa55udp: dial %s: %w", cc.Host, err)
	}

	cfg, err := buildReadConfig(cc.Id, cc.PDU, cc.Register, cc.Count, cc.Offset)
	if err != nil {
		return nil, err
	}

	return &AA55UDP{
		log:      util.NewLogger("aa55udp"),
		conn:     conn,
		pdu:      cfg.pdu,
		offset:   cfg.offset,
		decode:   cc.Decode,
		scale:    cc.Scale,
		useCache: cfg.useCache,
	}, nil
}

// FloatGetter implements the evcc Plugin interface.
func (p *AA55UDP) FloatGetter() (func() (float64, error), error) {
	return p.query, nil
}

// query fetches the payload and returns the decoded, scaled value at p.offset.
func (p *AA55UDP) query() (float64, error) {
	payload, err := p.fetch()
	if err != nil {
		return 0, err
	}

	minLen := p.offset + aa55.DecodeSize(p.decode)
	if len(payload) < minLen {
		return 0, fmt.Errorf("aa55udp: payload too short (len=%d, need=%d)", len(payload), minLen)
	}

	v, err := aa55.DecodeAt(payload, p.offset, p.decode)
	if err != nil {
		return 0, fmt.Errorf("aa55udp: %w", err)
	}

	return v * p.scale, nil
}

// fetch returns the response payload, using caching for block-read mode.
func (p *AA55UDP) fetch() ([]byte, error) {
	packet := append(p.pdu, aa55.ModbusCRC16(p.pdu)...)

	// No caching: direct send/recv
	if !p.useCache {
		raw, err := p.sendRecv(packet)
		if err != nil {
			return nil, err
		}
		payload, err := aa55.StripHeader(raw)
		if err != nil {
			return nil, fmt.Errorf("aa55udp: %w", err)
		}
		return payload, nil
	}

	// Caching: shared reads by (addr, pdu)
	key := p.conn.RemoteAddr().String() + "/" + hex.EncodeToString(p.pdu)

	if payload, ok := aa55.Cache.Get(key); ok {
		p.log.TRACE.Printf("cache hit for %s pdu=%s", p.conn.RemoteAddr(), hex.EncodeToString(p.pdu))
		return payload, nil
	}

	raw, err := p.sendRecv(packet)
	if err != nil {
		return nil, err
	}

	payload, err := aa55.StripHeader(raw)
	if err != nil {
		return nil, fmt.Errorf("aa55udp: %w", err)
	}

	aa55.Cache.Put(key, payload)
	return payload, nil
}

// sendRecv sends packet over p.conn and returns the raw response bytes.
func (p *AA55UDP) sendRecv(packet []byte) ([]byte, error) {
	p.log.TRACE.Printf("send to %s: %s", p.conn.RemoteAddr(), hex.EncodeToString(packet))

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

	p.log.TRACE.Printf("recv from %s: %s", p.conn.RemoteAddr(), hex.EncodeToString(buf[:n]))

	return buf[:n], nil
}
