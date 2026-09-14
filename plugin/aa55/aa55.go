// Package aa55 implements the GoodWe WiFi AA55-over-UDP wire protocol used by
// the GoodWe inverter families (DT/DNS, ES/EM, ET/EH/BT/BH).
//
// The inverter speaks a simple request/response protocol over UDP port 8899:
//
//	Request:  [6-byte PDU body] [Modbus CRC-16, little-endian]
//	Response: AA 55 [src] 03 [byteCount] [payload…] [CRC]
package aa55

import (
	"errors"
	"fmt"

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

// buildWriteSinglePDU constructs the 6-byte PDU body for a WRITE SINGLE REGISTER
// request: [addr, 0x06, reg_hi, reg_lo, val_hi, val_lo].
func buildWriteSinglePDU(addr byte, register, value uint16) []byte {
	return []byte{
		addr, modbus.FuncCodeWriteSingleRegister,
		byte(register >> 8), byte(register),
		byte(value >> 8), byte(value),
	}
}

// buildWriteMultiplePDU constructs the WRITE MULTIPLE REGISTERS PDU body:
// [addr, 0x10, reg_hi, reg_lo, count_hi, count_lo, byteCount, data…].
func buildWriteMultiplePDU(addr byte, register uint16, values []byte) []byte {
	count := uint16(len(values) / 2)
	pdu := []byte{
		addr, modbus.FuncCodeWriteMultipleRegisters,
		byte(register >> 8), byte(register),
		byte(count >> 8), byte(count),
		byte(len(values)),
	}
	return append(pdu, values...)
}

// validateWriteResponse checks the AA55 frame echoed for a write request:
// only the magic bytes and echoed function code are validated (high bit = reject).
func validateWriteResponse(buf []byte, funcCode byte) error {
	if len(buf) < 4 || buf[0] != 0xAA || buf[1] != 0x55 {
		return errors.New("invalid response header")
	}
	if buf[3] == funcCode|0x80 {
		return fmt.Errorf("write rejected (code %#x)", buf[3])
	}
	if buf[3] != funcCode {
		return fmt.Errorf("unexpected function code %#x", buf[3])
	}
	return nil
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
