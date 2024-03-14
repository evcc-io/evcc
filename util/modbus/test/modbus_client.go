package test

import (
	"encoding/binary"
	"fmt"

	gridx "github.com/grid-x/modbus"
)

var _ gridx.Client = (*ModbusTestClient)(nil)

type ModbusTestClient struct {
	OnReadFn  func(address, quantity uint16) (results []byte, err error)
	OnWriteFn func(address uint16, value []byte) (results []byte, err error)
}

func (m *ModbusTestClient) ReadCoils(address, quantity uint16) (results []byte, err error) {
	if m.OnReadFn == nil {
		return nil, fmt.Errorf("OnReadFn not set")
	}
	return m.OnReadFn(address, quantity)
}

func (m *ModbusTestClient) ReadDiscreteInputs(address, quantity uint16) (results []byte, err error) {
	return m.ReadCoils(address, quantity)
}

func (m *ModbusTestClient) WriteSingleCoil(address, value uint16) (results []byte, err error) {
	if m.OnWriteFn == nil {
		return nil, fmt.Errorf("OnWriteFn not set")
	}
	return m.OnWriteFn(address, binary.LittleEndian.AppendUint16(nil, value))
}

func (m *ModbusTestClient) WriteMultipleCoils(address, quantity uint16, value []byte) (results []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *ModbusTestClient) ReadInputRegisters(address, quantity uint16) (results []byte, err error) {
	return m.ReadCoils(address, quantity)
}

func (m *ModbusTestClient) ReadHoldingRegisters(address, quantity uint16) (results []byte, err error) {
	return m.ReadCoils(address, quantity)
}

func (m *ModbusTestClient) WriteSingleRegister(address, value uint16) (results []byte, err error) {
	return m.WriteSingleCoil(address, value)
}

func (m *ModbusTestClient) WriteMultipleRegisters(address, quantity uint16, value []byte) (results []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *ModbusTestClient) ReadWriteMultipleRegisters(readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) (results []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *ModbusTestClient) MaskWriteRegister(address, andMask, orMask uint16) (results []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *ModbusTestClient) ReadFIFOQueue(address uint16) (results []byte, err error) {
	//TODO implement me
	panic("implement me")
}
