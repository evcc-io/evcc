package plugin

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/evcc-io/evcc/util"
)

// AA55UDP implements the GoodWe WiFi AA55-over-UDP wire protocol as a generic
// evcc source plugin.
//
// The inverter speaks a simple request/response protocol over UDP port 8899:
//
//	Request:  [6-byte PDU body] [Modbus CRC-16, little-endian]
//	Response: AA 55 [src] 03 [byteCount] [payload…] [CRC]
//
// Two read modes are supported, selected at construction time:
//
//	Register read: single register, value at offset 0 of response payload.
//	Block read:    whole register block, value extracted at a byte offset.
//	               Multiple source blocks sharing the same (host, pdu) pair
//	               share one UDP exchange per poll cycle via a response cache.
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
//	host:     192.168.1.26
//	pdu:      "f703891c007d"  # 6-byte PDU hex including inverter address byte
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

	if err := validateDecode(cc.Decode); err != nil {
		return nil, err
	}

	raddr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(cc.Host, "8899"))
	if err != nil {
		return nil, fmt.Errorf("aa55udp: resolve %s: %w", cc.Host, err)
	}
	conn, err := net.DialUDP("udp4", nil, raddr)
	if err != nil {
		return nil, fmt.Errorf("aa55udp: dial %s: %w", cc.Host, err)
	}

	p := &AA55UDP{
		log:    util.NewLogger("aa55udp"),
		conn:   conn,
		decode: cc.Decode,
		scale:  cc.Scale,
	}

	if cc.PDU != "" {
		// Block mode: PDU-based read with caching.
		if cc.Register != 0 || cc.Count != 2 || cc.Id != int(aa55InverterAddr) {
			return nil, errors.New("aa55udp: pdu cannot be combined with register/count/id settings")
		}
		if cc.Offset < 0 {
			return nil, fmt.Errorf("aa55udp: offset must be non-negative, got %d", cc.Offset)
		}
		pdu, err := buildPDUFromHex(cc.PDU)
		if err != nil {
			return nil, err
		}
		p.pdu = pdu
		p.offset = cc.Offset
		p.useCache = true
	} else {
		// Register mode: id/register/count-based read without caching.
		if cc.Count == 0 {
			return nil, errors.New("aa55udp: count must be ≥ 1")
		}
		if cc.Id < 0 || cc.Id > 255 {
			return nil, fmt.Errorf("aa55udp: id must be 0-255, got %d", cc.Id)
		}
		p.pdu = buildPDU(byte(cc.Id), cc.Register, cc.Count)
		p.offset = 0
		p.useCache = false
	}

	return p, nil
}

// validateDecode returns an error if decode is not a supported type.
func validateDecode(decode string) error {
	switch decode {
	case "int32be", "uint32be", "uint32nan", "int16be", "uint16be", "float32be":
		return nil
	}
	return fmt.Errorf("aa55udp: unsupported decode %q (want int32be|uint32be|uint32nan|int16be|uint16be|float32be)", decode)
}

// decodeSize returns the number of bytes required to decode the given type.
func decodeSize(decode string) int {
	switch decode {
	case "float32be", "int32be", "uint32be", "uint32nan":
		return 4
	case "int16be", "uint16be":
		return 2
	}
	return 0
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

	minLen := p.offset + decodeSize(p.decode)
	if len(payload) < minLen {
		return 0, fmt.Errorf("aa55udp: payload too short (len=%d, need=%d)", len(payload), minLen)
	}

	v, err := decodeAt(payload, p.offset, p.decode)
	if err != nil {
		return 0, fmt.Errorf("aa55udp: %w", err)
	}

	return v * p.scale, nil
}

// fetch returns the response payload, using caching for block mode.
func (p *AA55UDP) fetch() ([]byte, error) {
	if p.useCache {
		return p.fetchWithCache()
	}
	return p.fetchDirect()
}

// fetchDirect performs a direct UDP request/response exchange without caching.
func (p *AA55UDP) fetchDirect() ([]byte, error) {
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

// fetchWithCache performs a UDP request/response with response caching for shared reads.
func (p *AA55UDP) fetchWithCache() ([]byte, error) {
	key := p.conn.RemoteAddr().String() + "/" + hex.EncodeToString(p.pdu)

	if payload, ok := responseCache.get(key); ok {
		pduHex := hex.EncodeToString(p.pdu)
		p.log.TRACE.Printf("cache hit for %s pdu=%s", p.conn.RemoteAddr(), pduHex)
		return payload, nil
	}

	packet := append(p.pdu, modbusCRC16(p.pdu)...)
	raw, err := p.sendRecv(packet)
	if err != nil {
		return nil, err
	}

	payload, err := stripAA55Header(raw)
	if err != nil {
		return nil, fmt.Errorf("aa55udp: %w", err)
	}

	responseCache.put(key, payload)
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

// stripAA55Header validates the AA55 response frame and returns the bare
// payload (without the 5-byte header and trailing 2-byte CRC).
// buf[2] is the inverter source address, which varies by family — only the
// AA 55 magic bytes and function code 0x03 are validated.
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

// decodeAt extracts a value at the given byte offset of payload and
// interprets it according to decode.
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
		// Used for PV string power registers where disconnected strings report NaN.
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
