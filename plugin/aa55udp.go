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
	"time"

	"github.com/evcc-io/evcc/util"
)

// AA55UDP implements the GoodWe WiFi AA55-over-UDP wire protocol as a generic
// evcc source plugin.
//
// The inverter speaks a simple request/response protocol over UDP port 8899:
//
//	Request:  [6-byte Modbus PDU body] [Modbus CRC-16, little-endian]
//	Response: AA 55 [src] 03 [byteCount] [payload…] [CRC]
//
// src varies by inverter family (0x7F for DT/DNS, 0xF7 for ET/EH/BT/BH);
// only the AA 55 magic bytes and function code 0x03 are validated.
//
// Each instance reads exactly one value from one register (or register pair
// for 32-bit values), matching how Modbus source plugins work.  The PDU is
// constructed from register and count; the decoded value is always at offset 0
// of the response payload.
type AA55UDP struct {
	log    *util.Logger
	conn   *net.UDPConn
	pdu    []byte // 6-byte PDU body, no CRC
	decode string // int32be | uint32be | int16be | uint16be | float32be
	scale  float64
}

func init() {
	registry.AddCtx("aa55udp", NewAA55UDPFromConfig)
}

// NewAA55UDPFromConfig creates an AA55UDP plugin from a source block:
//
//	source:   aa55udp
//	host:     192.168.1.26   # inverter IP; port 8899 is always used
//	id:       0x7F           # inverter address byte: 0x7F for DT/DNS/ES/EM, 0xF7 for ET/EH/BT/BH
//	register: 30127          # Modbus register address (0-based, uint16)
//	count:    2              # number of registers to read (1=U16, 2=S32/U32)
//	decode:   int32be        # int32be | uint32be | int16be | uint16be | float32be
//	scale:    1.0            # optional multiplier (default 1.0)
func NewAA55UDPFromConfig(_ context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		Host     string  `mapstructure:"host"`
		Id       int     `mapstructure:"id"`
		Register uint16  `mapstructure:"register"`
		Count    uint16  `mapstructure:"count"`
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

	if cc.Count == 0 {
		return nil, errors.New("aa55udp: count must be ≥ 1")
	}

	if cc.Id < 0 || cc.Id > 255 {
		return nil, fmt.Errorf("aa55udp: id must be 0-255, got %d", cc.Id)
	}

	switch cc.Decode {
	case "int32be", "uint32be", "int16be", "uint16be", "float32be":
	default:
		return nil, fmt.Errorf("aa55udp: unsupported decode %q (want int32be|uint32be|int16be|uint16be|float32be)", cc.Decode)
	}

	pdu := buildPDU(byte(cc.Id), cc.Register, cc.Count)

	addr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(cc.Host, "8899"))
	if err != nil {
		return nil, fmt.Errorf("aa55udp: resolve %s: %w", cc.Host, err)
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("aa55udp: dial %s: %w", cc.Host, err)
	}

	return &AA55UDP{
		log:    util.NewLogger("aa55udp"),
		conn:   conn,
		pdu:    pdu,
		decode: cc.Decode,
		scale:  cc.Scale,
	}, nil
}

// FloatGetter implements the evcc Plugin interface.
func (p *AA55UDP) FloatGetter() (func() (float64, error), error) {
	return p.query, nil
}

// query sends the PDU and returns the decoded, scaled value at offset 0 of the
// response payload.
func (p *AA55UDP) query() (float64, error) {
	packet := append(p.pdu, modbusCRC16(p.pdu)...)

	if _, err := p.conn.Write(packet); err != nil {
		return 0, fmt.Errorf("aa55udp write: %w", err)
	}

	if err := p.conn.SetReadDeadline(time.Now().Add(4 * time.Second)); err != nil {
		return 0, fmt.Errorf("aa55udp deadline: %w", err)
	}

	buf := make([]byte, 512)
	n, err := p.conn.Read(buf)
	if err != nil {
		return 0, fmt.Errorf("aa55udp read: %w", err)
	}

	payload, err := stripAA55Header(buf[:n])
	if err != nil {
		return 0, fmt.Errorf("aa55udp: %w", err)
	}

	v, err := decodeAt(payload, 0, p.decode)
	if err != nil {
		return 0, fmt.Errorf("aa55udp: %w", err)
	}

	return v * p.scale, nil
}

// aa55InverterAddr is the default inverter address byte, used by DT/DNS and ES/EM families.
// ET/EH/BT/BH families require 0xF7 instead.
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
// Kept for use in tests.
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
// buf[2] is the inverter source address and varies by family — only the
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

// decodeAt extracts an integer at the given byte offset of payload and
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
		for range 8 {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return []byte{byte(crc & 0xFF), byte(crc >> 8)}
}
