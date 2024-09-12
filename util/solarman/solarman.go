package solarman

import (
	"encoding/binary"
	"fmt"
	"math/rand"
)

type Settings struct {
	Host         string
	Port         int
	Loggerserial uint32
	Slaveid      byte
}

type Connection struct {
	conn         SolarmanConnection
	LoggerSerial uint32
	Serial       byte
	SlaveID      byte
}

func NewConnection(URI string, LoggerSerial uint32, SlaveID byte) (*Connection, error) {
	conn := GetConnection(URI)

	connection := &Connection{
		conn:         conn,
		SlaveID:      SlaveID,
		LoggerSerial: LoggerSerial,
		Serial:       byte(rand.Intn(0xFF)),
	}

	return connection, nil
}

func (c *Connection) requestData(request []byte) ([]byte, error) {
	data, err := c.conn.Send(request)

	if err != nil {
		return nil, err
	}

	dataLength := len(data)

	if data[0] != 0xA5 {
		return nil, fmt.Errorf("wrong start value")
	}
	if data[dataLength-1] != 0x15 {
		return nil, fmt.Errorf("wrong end value")
	}
	checksum := checkSum(data[1 : dataLength-2])
	if checksum != data[dataLength-2] {
		return nil, fmt.Errorf("wrong checksum")
	}

	if c.Serial != data[5] {
		return nil, fmt.Errorf("wrong serial number")
	}

	loggerSerial := binary.LittleEndian.Uint32(data[7:11])
	if loggerSerial != c.LoggerSerial {
		return nil, fmt.Errorf("logger serial does not match")
	}

	if data[3] != 0x10 && data[4] != 0x15 {
		return nil, fmt.Errorf("wrong control code")
	}

	if data[11] != 0x02 {
		return nil, fmt.Errorf("wrong frametype")
	}

	payloadLength := binary.LittleEndian.Uint16(data[1:3])

	if dataLength != int(payloadLength)+13 {
		return nil, fmt.Errorf("expected length does not fit real length of returned data")
	}

	modbusData := data[25 : dataLength-2]
	if len(modbusData) < 5 {
		return nil, fmt.Errorf("got invalid modbus data (too short)")
	}

	count := int(modbusData[2])
	return modbusData[3 : count+3], nil
}

func (c *Connection) exec(mapper func(*SolarmanRequestBuilder) []byte) ([]byte, error) {
	builder := NewSolarmanRequestBuilder(c.SlaveID, c.Serial).
		SetLoggerSerial(c.LoggerSerial)
	request := mapper(builder)

	return c.requestData(request)
}

func (c *Connection) ReadCoils(address uint16, quantity uint16) ([]byte, error) {
	return c.exec(func(srb *SolarmanRequestBuilder) []byte {
		return srb.ReadCoilsRequest(address, quantity)
	})
}

func (c *Connection) ReadHoldingRegisters(address uint16, quantity uint16) ([]byte, error) {
	return c.exec(func(srb *SolarmanRequestBuilder) []byte {
		return srb.ReadHoldingRegistersRequest(address, quantity)
	})
}

func (c *Connection) ReadInputRegisters(address uint16, quantity uint16) ([]byte, error) {
	return c.exec(func(srb *SolarmanRequestBuilder) []byte {
		return srb.ReadInputRegistersRequest(address, quantity)
	})
}

func (c *Connection) WriteSingleCoil(address uint16, value uint16) ([]byte, error) {
	return c.exec(func(srb *SolarmanRequestBuilder) []byte {
		return srb.WriteSingleCoilRequest(address, value)
	})
}

func (c *Connection) WriteSingleRegister(address uint16, value uint16) ([]byte, error) {
	return c.exec(func(srb *SolarmanRequestBuilder) []byte {
		return srb.WriteSingleRegisterRequest(address, value)
	})
}

func (c *Connection) WriteMultipleRegisters(address uint16, quantity uint16, values []byte) ([]byte, error) {
	return c.exec(func(srb *SolarmanRequestBuilder) []byte {
		return srb.WriteMultipleRegistersRquest(address, quantity, values)
	})
}
