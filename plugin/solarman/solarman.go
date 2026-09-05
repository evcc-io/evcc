// Package solarman implements the Solarman V5 TCP transport.
package solarman

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	startByte             = 0xA5
	endByte               = 0x15
	requestControlCode    = 0x45
	responseControlCode   = 0x15
	responseModbusOffset  = 25
	maxFramePayloadLength = 1024
)

// Client reads Modbus registers through a Solarman V5 logger.
type Client struct {
	address string
	serial  uint32
	timeout time.Duration

	mu       sync.Mutex
	sequence uint8
}

// New creates a Solarman V5 client.
func New(host string, port int, serial uint32, timeout time.Duration) (*Client, error) {
	if host == "" {
		return nil, errors.New("missing host")
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}
	if timeout <= 0 {
		return nil, errors.New("invalid timeout")
	}

	sequence := []byte{1}
	if _, err := rand.Read(sequence); err == nil {
		sequence[0] = sequence[0]%254 + 1
	}

	return &Client{
		address:  net.JoinHostPort(host, strconv.Itoa(port)),
		serial:   serial,
		timeout:  timeout,
		sequence: sequence[0],
	}, nil
}

// ReadHoldingRegisters reads holding registers from the connected inverter.
func (c *Client) ReadHoldingRegisters(ctx context.Context, id byte, address, count uint16) ([]byte, error) {
	return c.readRegisters(ctx, id, 0x03, address, count)
}

// ReadInputRegisters reads input registers from the connected inverter.
func (c *Client) ReadInputRegisters(ctx context.Context, id byte, address, count uint16) ([]byte, error) {
	return c.readRegisters(ctx, id, 0x04, address, count)
}

func (c *Client) readRegisters(ctx context.Context, id, function byte, address, count uint16) ([]byte, error) {
	if c.serial == 0 {
		return nil, errors.New("missing logger serial")
	}
	if count == 0 || count > 125 {
		return nil, fmt.Errorf("invalid register count: %d", count)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", c.address)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", c.address, err)
	}
	defer conn.Close()

	deadline := time.Now().Add(c.timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("set deadline: %w", err)
	}

	sequence := c.sequence
	c.sequence++

	if _, err := conn.Write(request(c.serial, sequence, id, function, address, count)); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	for {
		frame, err := readFrame(conn)
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		if frame[5] != sequence {
			continue
		}

		response, err := validateResponse(frame, c.serial, sequence)
		if err != nil {
			return nil, err
		}

		return validateModbusResponse(response, id, function, count)
	}
}

func request(serial uint32, sequence, id, function byte, address, count uint16) []byte {
	modbus := []byte{id, function, byte(address >> 8), byte(address), byte(count >> 8), byte(count)}
	modbus = append(modbus, crc(modbus)...)

	payload := make([]byte, 15, 15+len(modbus))
	payload[0] = 0x02
	payload = append(payload, modbus...)

	frame := []byte{startByte, byte(len(payload)), byte(len(payload) >> 8), 0x10, requestControlCode, sequence, 0x00}
	serialBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(serialBytes, serial)
	frame = append(frame, serialBytes...)
	frame = append(frame, payload...)
	frame = append(frame, checksum(frame[1:]), endByte)

	return frame
}

func readFrame(r io.Reader) ([]byte, error) {
	header := make([]byte, 3)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	if header[0] != startByte {
		return nil, fmt.Errorf("invalid frame start: %02x", header[0])
	}

	payloadLength := int(binary.LittleEndian.Uint16(header[1:]))
	if payloadLength > maxFramePayloadLength {
		return nil, fmt.Errorf("frame payload too large: %d", payloadLength)
	}

	frame := make([]byte, 3+payloadLength+10)
	copy(frame, header)
	if _, err := io.ReadFull(r, frame[3:]); err != nil {
		return nil, err
	}

	return frame, nil
}

func validateResponse(frame []byte, serial uint32, sequence byte) ([]byte, error) {
	if len(frame) < responseModbusOffset+5 {
		return nil, errors.New("response frame too short")
	}
	if frame[len(frame)-1] != endByte {
		return nil, fmt.Errorf("invalid frame end: %02x", frame[len(frame)-1])
	}
	if frame[len(frame)-2] != checksum(frame[1:len(frame)-2]) {
		return nil, errors.New("invalid frame checksum")
	}
	if frame[3] != 0x10 || frame[4] != responseControlCode {
		return nil, fmt.Errorf("unexpected response control code: %02x%02x", frame[4], frame[3])
	}
	if frame[5] != sequence {
		return nil, fmt.Errorf("unexpected response sequence: %d", frame[5])
	}
	if binary.LittleEndian.Uint32(frame[7:11]) != serial {
		return nil, errors.New("unexpected logger serial")
	}
	if frame[11] != 0x02 {
		return nil, fmt.Errorf("unexpected response frame type: %02x", frame[11])
	}

	return frame[responseModbusOffset : len(frame)-2], nil
}

func validateModbusResponse(response []byte, id, function byte, count uint16) ([]byte, error) {
	if len(response) >= 2 && response[len(response)-1] == 0 && response[len(response)-2] == 0 && validCRC(response[:len(response)-2]) {
		response = response[:len(response)-2]
	}
	if len(response) < 5 {
		return nil, errors.New("modbus response too short")
	}
	if !validCRC(response) {
		return nil, errors.New("invalid modbus checksum")
	}
	if response[0] != id {
		return nil, fmt.Errorf("unexpected modbus id: %d", response[0])
	}
	if response[1] == function|0x80 {
		return nil, fmt.Errorf("modbus exception: %d", response[2])
	}
	if response[1] != function {
		return nil, fmt.Errorf("unexpected modbus function: %d", response[1])
	}

	byteCount := int(response[2])
	if byteCount != int(count)*2 || len(response) != byteCount+5 {
		return nil, fmt.Errorf("unexpected modbus response length: %d", byteCount)
	}

	return response[3 : 3+byteCount], nil
}

func checksum(data []byte) byte {
	var sum byte
	for _, value := range data {
		sum += value
	}
	return sum
}

func validCRC(data []byte) bool {
	if len(data) < 3 {
		return false
	}

	want := binary.LittleEndian.Uint16(data[len(data)-2:])
	got := binary.LittleEndian.Uint16(crc(data[:len(data)-2]))

	return got == want
}

func crc(data []byte) []byte {
	value := uint16(0xFFFF)
	for _, current := range data {
		value ^= uint16(current)
		for bit := 0; bit < 8; bit++ {
			if value&1 == 1 {
				value = value>>1 ^ 0xA001
			} else {
				value >>= 1
			}
		}
	}

	return []byte{byte(value), byte(value >> 8)}
}
