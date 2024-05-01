package modbus

import "github.com/volkszaehler/mbmd/meters"

// Connection is a logical modbus connection per slave ID sharing a physical connection
type Connection struct {
	meters.Connection
}

func (mb *Connection) ReadCoils(address, quantity uint16) ([]byte, error) {
	return mb.ModbusClient().ReadCoils(address, quantity)
}

func (mb *Connection) WriteSingleCoil(address, value uint16) ([]byte, error) {
	return mb.ModbusClient().WriteSingleCoil(address, value)
}

func (mb *Connection) ReadInputRegisters(address, quantity uint16) ([]byte, error) {
	return mb.ModbusClient().ReadInputRegisters(address, quantity)
}

func (mb *Connection) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	return mb.ModbusClient().ReadHoldingRegisters(address, quantity)
}

func (mb *Connection) WriteSingleRegister(address, value uint16) ([]byte, error) {
	return mb.ModbusClient().WriteSingleRegister(address, value)
}

func (mb *Connection) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	return mb.ModbusClient().WriteMultipleRegisters(address, quantity, value)
}

func (mb *Connection) ReadDiscreteInputs(address, quantity uint16) (results []byte, err error) {
	return mb.ModbusClient().ReadDiscreteInputs(address, quantity)
}

func (mb *Connection) WriteMultipleCoils(address, quantity uint16, value []byte) (results []byte, err error) {
	return mb.ModbusClient().WriteMultipleCoils(address, quantity, value)
}

func (mb *Connection) ReadWriteMultipleRegisters(readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) (results []byte, err error) {
	return mb.ModbusClient().ReadWriteMultipleRegisters(readAddress, readQuantity, writeAddress, writeQuantity, value)
}

func (mb *Connection) MaskWriteRegister(address, andMask, orMask uint16) (results []byte, err error) {
	return mb.ModbusClient().MaskWriteRegister(address, andMask, orMask)
}

func (mb *Connection) ReadFIFOQueue(address uint16) (results []byte, err error) {
	return mb.ModbusClient().ReadFIFOQueue(address)
}

func (c *Connection) Clone(slaveID uint8) *Connection {
	conn := *c
	conn.Connection = c.Connection.Clone(slaveID)
	return &conn
}
