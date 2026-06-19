package aa55

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// cacheTTL serves all sources within one poll cycle (well under 1s) while
// forcing a fresh read on the next cycle.
const cacheTTL = 2 * time.Second

// cache de-duplicates block reads across all AA55UDP instances so multiple
// sources covering the same (host, block) share one UDP exchange per cycle.
var cache = modbus.NewCache(cacheTTL)

// AA55UDP is the GoodWe AA55-over-UDP source plugin transport.
//
// Two read modes are supported, both built from logical parameters
// (id, register, count) on the Go side:
//
//	Register read: single register, value at offset 0 of response payload.
//	Block read:    enclosing block fetched once, target register extracted at
//	               its computed offset. Multiple sources sharing the same
//	               (host, block) share one UDP exchange per poll cycle via the
//	               response cache.
type AA55UDP struct {
	log      *util.Logger
	conn     *net.UDPConn
	pdu      []byte               // 6-byte PDU body, no CRC
	offset   int                  // byte offset into the response payload (0 for register reads)
	length   int                  // value length in bytes
	decode   func([]byte) float64 // modbus register decoder
	scale    float64
	cacheKey string        // precomputed cache key (remoteAddr/pdu); empty disables caching
	delay    time.Duration // minimum gap between sends to the inverter (0 disables)
}

// readConfig holds the resolved read mode configuration.
type readConfig struct {
	pdu      []byte
	offset   int
	useCache bool
}

// buildReadConfig resolves the read mode from the target register (register,
// count, id) and the optional enclosing block. In both modes the PDU is built
// on the Go side; the template only supplies logical parameters.
func buildReadConfig(id int, register, count uint16, block *modbus.Block) (readConfig, error) {
	if id < 0 || id > 255 {
		return readConfig{}, fmt.Errorf("id must be 0-255, got %d", id)
	}
	if count == 0 {
		return readConfig{}, errors.New("count must be ≥ 1")
	}

	// Block mode: fetch the whole block and extract the target register at its
	// offset. Multiple sources sharing the same block share one UDP exchange.
	if block != nil {
		if block.Count == 0 {
			return readConfig{}, errors.New("block count must be ≥ 1")
		}
		if !block.Contains(register, count) {
			return readConfig{}, fmt.Errorf("register %d+%d does not fit in block %d+%d", register, count, block.Register, block.Count)
		}
		return readConfig{
			pdu:      buildPDU(byte(id), block.Register, block.Count),
			offset:   block.ByteOffset(register),
			useCache: true,
		}, nil
	}

	// Register mode: single targeted read, value at offset 0, no caching.
	return readConfig{pdu: buildPDU(byte(id), register, count), useCache: false}, nil
}

// New constructs an AA55UDP from a high-level configuration. The register count
// and decoder are derived from reg; the read mode is resolved from block.
func New(log *util.Logger, conn *net.UDPConn, id int, reg modbus.Register, block *modbus.Block, scale float64, delay time.Duration) (*AA55UDP, error) {
	count, err := reg.Length()
	if err != nil {
		return nil, err
	}
	decode, err := reg.DecodeFunc()
	if err != nil {
		return nil, err
	}
	cfg, err := buildReadConfig(id, reg.Address, count, block)
	if err != nil {
		return nil, err
	}
	ap := &AA55UDP{
		log:    log,
		conn:   conn,
		decode: decode,
		length: int(count) * 2,
		scale:  scale,
		delay:  delay,
		pdu:    cfg.pdu,
		offset: cfg.offset,
	}
	if cfg.useCache {
		ap.cacheKey = conn.RemoteAddr().String() + "/" + string(cfg.pdu)
	}
	return ap, nil
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

	end := p.offset + p.length
	if len(payload) < end {
		return 0, fmt.Errorf("payload too short (len=%d, need=%d)", len(payload), end)
	}

	return p.decode(payload[p.offset:end]) * p.scale, nil
}

// fetch returns the response payload. In block-read mode the shared cache
// serves and de-duplicates requests.
func (p *AA55UDP) fetch() ([]byte, error) {
	// Register mode: single targeted read, no caching.
	if p.cacheKey == "" {
		return p.exchange()
	}

	payload, ok, err := cache.Fetch(p.cacheKey, p.exchange)
	if err != nil {
		return nil, err
	}
	if ok {
		p.log.TRACE.Printf("cache hit for %s pdu=%x", p.conn.RemoteAddr(), p.pdu)
	}
	return payload, nil
}

// exchange performs one request/response round trip and returns the response
// payload with the AA55 header stripped. It is the cache-miss path shared by
// single flight in block-read mode.
func (p *AA55UDP) exchange() ([]byte, error) {
	packet := append(p.pdu, modbusCRC16(p.pdu)...)
	raw, err := p.sendRecv(packet)
	if err != nil {
		return nil, err
	}
	return stripHeader(raw)
}

// sendRecv sends packet over p.conn and returns the raw response bytes. When a
// delay is set it serializes and spaces exchanges to the same inverter.
func (p *AA55UDP) sendRecv(packet []byte) ([]byte, error) {
	if p.delay > 0 {
		g := pace.gate(p.conn.RemoteAddr().String())
		g.mu.Lock()
		defer g.mu.Unlock()
		g.wait(p.delay)
	}

	p.log.TRACE.Printf("send %s: %x", p.conn.RemoteAddr(), packet)

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

	p.log.TRACE.Printf("recv %s: %x", p.conn.RemoteAddr(), buf[:n])

	return buf[:n], nil
}
