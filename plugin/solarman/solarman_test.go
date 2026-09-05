package solarman

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"testing"
	"time"
)

func TestReadHoldingRegisters(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	const serial = 3875738533
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()

		frame, err := readFrame(conn)
		if err != nil {
			done <- err
			return
		}
		if got := binary.LittleEndian.Uint32(frame[7:11]); got != serial {
			done <- errors.New("request has wrong logger serial")
			return
		}
		modbus := frame[26 : len(frame)-2]
		want := []byte{1, 3, 0, 86, 0, 2}
		if string(modbus[:6]) != string(want) || !validCRC(modbus) {
			done <- errors.New("request has wrong Modbus payload")
			return
		}

		response := []byte{1, 3, 4, 0, 0, 2, 48}
		response = append(response, crc(response)...)
		response = append(response, 0, 0)
		if _, err := conn.Write(responseFrame(serial, frame[5]+1, response)); err != nil {
			done <- err
			return
		}
		_, err = conn.Write(responseFrame(serial, frame[5], response))
		done <- err
	}()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	portNumber, err := net.LookupPort("tcp", port)
	if err != nil {
		t.Fatal(err)
	}

	client, err := New(host, portNumber, serial, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	value, err := client.ReadHoldingRegisters(context.Background(), 1, 86, 2)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{0, 0, 2, 48}; string(value) != string(want) {
		t.Fatalf("value: got %x, want %x", value, want)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestValidateResponseChecksum(t *testing.T) {
	frame := responseFrame(1, 1, []byte{1, 3, 2, 0, 1, 0, 0})
	frame[len(frame)-2]++

	if _, err := validateResponse(frame, 1, 1); err == nil {
		t.Fatal("expected checksum error")
	}
}

func responseFrame(serial uint32, sequence byte, modbus []byte) []byte {
	payload := make([]byte, 14, 14+len(modbus))
	payload[0] = 0x02
	payload = append(payload, modbus...)

	frame := []byte{startByte, byte(len(payload)), byte(len(payload) >> 8), 0x10, responseControlCode, sequence, 0x00}
	serialBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(serialBytes, serial)
	frame = append(frame, serialBytes...)
	frame = append(frame, payload...)
	frame = append(frame, checksum(frame[1:]), endByte)

	return frame
}
