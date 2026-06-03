// Package aa55 implements the GoodWe WiFi AA55-over-UDP wire protocol used by
// the GoodWe inverter families (DT/DNS, ES/EM, ET/EH/BT/BH).
//
// The inverter speaks a simple request/response protocol over UDP port 8899:
//
//	Request:  [6-byte PDU body] [Modbus CRC-16, little-endian]
//	Response: AA 55 [src] 03 [byteCount] [payload…] [CRC]
package aa55

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/grid-x/modbus"
)

// InverterAddr is the default inverter address byte, used by DT/DNS and ES/EM
// families. ET/EH/BT/BH families require 0xF7 (247) instead.
const InverterAddr byte = 0x7F

// buildPDU constructs the 6-byte PDU body for a READ HOLDING REGISTERS request.
// addr is the inverter address byte: 0x7F for DT/DNS/ES/EM, 0xF7 for ET/EH/BT/BH.
func buildPDU(addr byte, register, count uint16) []byte {
	return []byte{
		addr, modbus.FuncCodeReadHoldingRegisters,
		byte(register >> 8), byte(register),
		byte(count >> 8), byte(count),
	}
}

// stripHeader validates the AA55 response frame and returns the bare payload
// (without the 5-byte header and trailing 2-byte CRC).
// buf[2] is the inverter source address, which varies by family — only the
// AA 55 magic bytes and the READ HOLDING REGISTERS function code are validated.
func stripHeader(buf []byte) ([]byte, error) {
	if len(buf) < 6 || buf[0] != 0xAA || buf[1] != 0x55 || buf[3] != modbus.FuncCodeReadHoldingRegisters {
		return nil, errors.New("invalid response header")
	}
	byteCount := int(buf[4])
	if len(buf) < 5+byteCount+2 {
		return nil, errors.New("short response")
	}
	return buf[5 : 5+byteCount], nil
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

// decodeMeta describes the properties of a supported decode type.
type decodeMeta struct {
	size int
}

func decodeMetadata(name string) (decodeMeta, bool) {
	switch name {
	case "float32be", "int32be", "uint32be", "uint32nan":
		return decodeMeta{size: 4}, true
	case "int16be", "uint16be":
		return decodeMeta{size: 2}, true
	default:
		return decodeMeta{}, false
	}
}

// validateDecode returns an error if decode is not a supported type.
func validateDecode(decode string) error {
	if _, ok := decodeMetadata(decode); !ok {
		return fmt.Errorf("unsupported decode %q (want int32be|uint32be|uint32nan|int16be|uint16be|float32be)", decode)
	}
	return nil
}

// decodeSize returns the number of bytes required to decode the given type.
// Panics if decode type is unknown — callers must validateDecode first.
func decodeSize(decode string) int {
	if info, ok := decodeMetadata(decode); ok {
		return info.size
	}
	panic(fmt.Sprintf("unknown decode type %q", decode))
}

// decodeAt extracts a value at the given byte offset of payload and interprets
// it according to decode.
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
