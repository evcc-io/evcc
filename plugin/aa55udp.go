package plugin

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net"

	"github.com/evcc-io/evcc/util"
)

// aa55Mode selects the fetch strategy used by query.
type aa55Mode int

const (
	modeRegister aa55Mode = iota // single-register read, value at offset 0
	modeBlock                    // block read, value extracted at explicit offset
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
	log           *util.Logger
	conn          *net.UDPConn
	pdu           []byte   // 6-byte PDU body, no CRC
	offset        int      // byte offset into the response payload (0 for register reads)
	decode        string   // int32be | uint32be | uint32nan | int16be | uint16be | float32be
	scale         float64
	mode          aa55Mode // modeRegister or modeBlock
	minPayloadLen int      // minimum required payload length (offset + decode size)
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
		Id:       int(aa55InverterAddr),
		Register: 0, // block mode requires register=0 (PDU supplies full register address)
		Count:    2,
		Scale:    1.0,
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
		if err := initBlockMode(p, cc.PDU, cc.Offset, cc.Register, cc.Count, cc.Id); err != nil {
			return nil, err
		}
	} else {
		if err := initRegisterMode(p, cc.Id, cc.Register, cc.Count); err != nil {
			return nil, err
		}
	}

	return p, nil
}

// initBlockMode configures p for block read mode.
func initBlockMode(p *AA55UDP, pduHex string, offset int, register uint16, count uint16, id int) error {
	// Reject mixed configuration where block-mode PDU and register parameters are both set.
	if register != 0 || count != 2 || id != int(aa55InverterAddr) {
		return errors.New("aa55udp: pdu cannot be combined with register/count/id settings")
	}
	if offset < 0 {
		return fmt.Errorf("aa55udp: offset must be non-negative, got %d", offset)
	}
	pdu, err := buildPDUFromHex(pduHex)
	if err != nil {
		return err
	}
	p.pdu = pdu
	p.offset = offset
	p.mode = modeBlock
	p.minPayloadLen = offset + decodeSize(p.decode)
	return nil
}

// initRegisterMode configures p for register read mode.
func initRegisterMode(p *AA55UDP, id int, register uint16, count uint16) error {
	if count == 0 {
		return errors.New("aa55udp: count must be ≥ 1")
	}
	if id < 0 || id > 255 {
		return fmt.Errorf("aa55udp: id must be 0-255, got %d", id)
	}
	p.pdu = buildPDU(byte(id), register, count)
	p.offset = 0
	p.mode = modeRegister
	p.minPayloadLen = decodeSize(p.decode)
	return nil
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

// query fetches the payload via the mode-appropriate method and returns the
// decoded, scaled value at p.offset.
func (p *AA55UDP) query() (float64, error) {
	var (
		payload []byte
		err     error
	)

	switch p.mode {
	case modeRegister:
		payload, err = p.fetchRegister()
	case modeBlock:
		payload, err = p.fetchBlock()
	default:
		err = fmt.Errorf("aa55udp: unknown mode %d", p.mode)
	}

	if err != nil {
		return 0, err
	}

	v, err := decodeAt(payload, p.offset, p.decode)
	if err != nil {
		return 0, fmt.Errorf("aa55udp: %w", err)
	}

	return v * p.scale, nil
}

// cacheKeyAndPDUHex returns the cache key and hex-encoded PDU for logging and cache operations.
func (p *AA55UDP) cacheKeyAndPDUHex() (string, string) {
	pduHex := hex.EncodeToString(p.pdu)
	key := p.conn.RemoteAddr().String() + "/" + pduHex
	return key, pduHex
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
