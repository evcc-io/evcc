package plugin

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
)

// responseCache caches block-read payloads keyed by "raddr/pdu_hex" with a
// short TTL.  This allows multiple aa55udp source blocks that read different
// offsets from the same block PDU (e.g. all four Ppv string registers) to
// share a single UDP exchange per poll cycle.
type cacheEntry struct {
	payload   []byte
	expiresAt time.Time
}

var (
	responseCache   = make(map[string]cacheEntry)
	responseCacheMu sync.Mutex
)

// responseCacheTTL must be long enough to cover all sequential source reads
// within one evcc poll cycle (typically < 1 s), but short enough that the
// next cycle fetches fresh data.
const responseCacheTTL = 5 * time.Second

// AA55UDP implements the GoodWe WiFi AA55-over-UDP wire protocol as a generic
// evcc source plugin.
//
// The inverter speaks a simple request/response protocol over UDP port 8899:
//
//	Request:  [6-byte PDU body] [Modbus CRC-16, little-endian]
//	Response: AA 55 [src] 03 [byteCount] [payload…] [CRC]
//
// Two addressing modes are supported:
//
//	Register mode (per-register read, offset always 0):
//	  id, register, count → plugin builds the 6-byte PDU
//
//	PDU mode (block read, explicit offset):
//	  pdu (hex), offset → plugin uses the literal PDU; multiple sources
//	  sharing the same (host, pdu) pair share one UDP exchange via the cache.
//
// src varies by inverter family (0x7F for DT/DNS/ES/EM, 0xF7 for ET/EH/BT/BH).
type AA55UDP struct {
	log    *util.Logger
	conn   *net.UDPConn
	raddr  *net.UDPAddr
	pdu    []byte // 6-byte PDU body, no CRC
	offset int    // byte offset into the response payload
	decode string // int32be | uint32be | uint32nan | int16be | uint16be | float32be
	scale  float64
}

func init() {
	registry.AddCtx("aa55udp", NewAA55UDPFromConfig)
}

// NewAA55UDPFromConfig creates an AA55UDP plugin.
//
// Register mode (per-register):
//
//	source:   aa55udp
//	host:     192.168.1.26
//	id:       127          # 0x7F for DT/DNS/ES/EM (default); 247 (0xF7) for ET/EH/BT/BH
//	register: 30127
//	count:    2            # 1 = 16-bit, 2 = 32-bit
//	decode:   int32be
//	scale:    1.0
//
// PDU mode (block read with offset):
//
//	source:   aa55udp
//	host:     192.168.1.26
//	pdu:      "f703891c007d"  # 6-byte PDU hex including address byte
//	offset:   78              # byte offset into the response payload
//	decode:   int32be
//	scale:    1.0
func NewAA55UDPFromConfig(_ context.Context, other map[string]interface{}) (Plugin, error) {
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
		Id:    int(aa55InverterAddr),
		Count: 2,
		Scale: 1.0,
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	switch cc.Decode {
	case "int32be", "uint32be", "uint32nan", "int16be", "uint16be", "float32be":
	default:
		return nil, fmt.Errorf("aa55udp: unsupported decode %q (want int32be|uint32be|uint32nan|int16be|uint16be|float32be)", cc.Decode)
	}

	// Build or parse the PDU.
	var pdu []byte
	var offset int

	if cc.PDU != "" {
		// PDU mode: explicit hex + offset.
		var err error
		pdu, err = parsePDUHex(cc.PDU)
		if err != nil {
			return nil, err
		}
		offset = cc.Offset
	} else {
		// Register mode: build PDU from id/register/count, offset always 0.
		if cc.Count == 0 {
			return nil, errors.New("aa55udp: count must be ≥ 1")
		}
		if cc.Id < 0 || cc.Id > 255 {
			return nil, fmt.Errorf("aa55udp: id must be 0-255, got %d", cc.Id)
		}
		pdu = buildPDU(byte(cc.Id), cc.Register, cc.Count)
		offset = 0
	}

	raddr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(cc.Host, "8899"))
	if err != nil {
		return nil, fmt.Errorf("aa55udp: resolve %s: %w", cc.Host, err)
	}

	conn, err := net.DialUDP("udp4", nil, raddr)
	if err != nil {
		return nil, fmt.Errorf("aa55udp: dial %s: %w", cc.Host, err)
	}

	return &AA55UDP{
		log:    util.NewLogger("aa55udp"),
		conn:   conn,
		raddr:  raddr,
		pdu:    pdu,
		offset: offset,
		decode: cc.Decode,
		scale:  cc.Scale,
	}, nil
}

// FloatGetter implements the evcc Plugin interface.
func (p *AA55UDP) FloatGetter() (func() (float64, error), error) {
	return p.query, nil
}

// query returns the decoded, scaled value.  The response payload is cached
// by (raddr, pdu) so that multiple sources sharing the same block PDU only
// trigger one UDP exchange per cache TTL window.
func (p *AA55UDP) query() (float64, error) {
	payload, err := p.fetchPayload()
	if err != nil {
		return 0, err
	}

	v, err := decodeAt(payload, p.offset, p.decode)
	if err != nil {
		return 0, fmt.Errorf("aa55udp: %w", err)
	}

	return v * p.scale, nil
}

// fetchPayload returns the response payload, using the cache when available.
func (p *AA55UDP) fetchPayload() ([]byte, error) {
	key := p.raddr.String() + "/" + hex.EncodeToString(p.pdu)

	responseCacheMu.Lock()
	if entry, ok := responseCache[key]; ok && time.Now().Before(entry.expiresAt) {
		payload := entry.payload
		responseCacheMu.Unlock()
		return payload, nil
	}
	responseCacheMu.Unlock()

	// Cache miss — send the request.
	packet := append(p.pdu, modbusCRC16(p.pdu)...)

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

	payload, err := stripAA55Header(buf[:n])
	if err != nil {
		return nil, fmt.Errorf("aa55udp: %w", err)
	}

	responseCacheMu.Lock()
	responseCache[key] = cacheEntry{payload: payload, expiresAt: time.Now().Add(responseCacheTTL)}
	responseCacheMu.Unlock()

	return payload, nil
}

// aa55InverterAddr is the default inverter address byte, used by DT/DNS and ES/EM families.
// ET/EH/BT/BH families require 0xF7 (247) instead.
const aa55InverterAddr = 0x7F

// aa55ReadFunc is the Modbus function code for READ HOLDING REGISTERS.
const aa55ReadFunc = 0x03

// buildPDU constructs the 6-byte PDU for a READ HOLDING REGISTERS request.
// addr is the inverter address byte: 0x7F for DT/DNS/ES/EM, 0xF7 for ET/EH/BT/BH.
func buildPDU(addr byte, register, count uint16) []byte {
	return []byte{
		addr, aa55ReadFunc,
		byte(register >> 8), byte(register),
		byte(count >> 8), byte(count),
	}
}

// parsePDUHex decodes a hex string (spaces allowed) into exactly 6 bytes.
func parsePDUHex(s string) ([]byte, error) {
	clean := strings.ReplaceAll(s, " ", "")
	b, err := hex.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("aa55udp: invalid pdu %q: %w", s, err)
	}
	if len(b) != 6 {
		return nil, fmt.Errorf("aa55udp: pdu must be 6 bytes, got %d", len(b))
	}
	return b, nil
}

// stripAA55Header validates the AA55 response frame and returns the bare
// payload (without the 5-byte header and trailing 2-byte CRC).
func stripAA55Header(buf []byte) ([]byte, error) {
	if len(buf) < 6 || buf[0] != 0xAA || buf[1] != 0x55 || buf[3] != 0x03 {
		return nil, errors.New("invalid response header")
	}
	byteCount := int(buf[4])
	if len(buf) < 5+byteCount+2 {
		return nil, errors.New("short response")
	}
	return buf[5 : 5+byteCount], nil
}

// decodeAt extracts a value at the given byte offset of payload.
func decodeAt(payload []byte, offset int, decode string) (float64, error) {
	switch decode {
	case "float32be":
		if len(payload) < offset+4 {
			return 0, fmt.Errorf("payload too short for float32be at offset %d (len=%d)", offset, len(payload))
		}
		bits := binary.BigEndian.Uint32(payload[offset:])
		return float64(math.Float32frombits(bits)), nil
	case "int32be":
		if len(payload) < offset+4 {
			return 0, fmt.Errorf("payload too short for int32be at offset %d (len=%d)", offset, len(payload))
		}
		return float64(int32(binary.BigEndian.Uint32(payload[offset:]))), nil
	case "uint32be":
		if len(payload) < offset+4 {
			return 0, fmt.Errorf("payload too short for uint32be at offset %d (len=%d)", offset, len(payload))
		}
		return float64(binary.BigEndian.Uint32(payload[offset:])), nil
	case "uint32nan":
		// Like uint32be but treats 0xFFFFFFFF (not-connected sentinel) as 0.
		if len(payload) < offset+4 {
			return 0, fmt.Errorf("payload too short for uint32nan at offset %d (len=%d)", offset, len(payload))
		}
		if v := binary.BigEndian.Uint32(payload[offset:]); v != 0xFFFFFFFF {
			return float64(v), nil
		}
		return 0, nil
	case "int16be":
		if len(payload) < offset+2 {
			return 0, fmt.Errorf("payload too short for int16be at offset %d (len=%d)", offset, len(payload))
		}
		return float64(int16(binary.BigEndian.Uint16(payload[offset:]))), nil
	case "uint16be":
		if len(payload) < offset+2 {
			return 0, fmt.Errorf("payload too short for uint16be at offset %d (len=%d)", offset, len(payload))
		}
		return float64(binary.BigEndian.Uint16(payload[offset:])), nil
	}
	return 0, fmt.Errorf("unknown decode type: %s", decode)
}

// modbusCRC16 computes the Modbus CRC-16 (little-endian byte order).
func modbusCRC16(data []byte) []byte {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return []byte{byte(crc & 0xFF), byte(crc >> 8)}
}
