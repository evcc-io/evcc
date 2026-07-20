package modbus

import (
	"fmt"
	"time"

	"github.com/volkszaehler/mbmd/meters"
)

// Connection is a logical modbus connection per slave ID sharing a physical connection
type Connection struct {
	*logger
	meters.Connection
	slaveID uint8 // duplicated from meters.Connection
	logical meters.Logger
	delay   time.Duration
}

func (c *Connection) Addr() string {
	return fmt.Sprintf("%s::%d", c.Connection.String(), c.slaveID)
}

func (c *Connection) Logger(logger meters.Logger) {
	c.logical = logger
}

func (c *Connection) Delay(delay time.Duration) {
	c.delay = delay
}

func (c *Connection) Clone(slaveID uint8) *Connection {
	return &Connection{
		slaveID:    slaveID,
		Connection: c.Connection.Clone(slaveID),
		logger:     c.logger,
	}
}

// TODO resolve conflicts
func (c *Connection) ConnectDelay(delay time.Duration) {
	if delay > 0 {
		c.Connection.ConnectDelay(delay)
	}
}

// TODO resolve conflicts
func (c *Connection) Timeout(timeout time.Duration) {
	if timeout > 0 {
		_ = c.Connection.Timeout(timeout)
	}
}

func (c *Connection) exec(fun func() ([]byte, error)) ([]byte, error) {
	return c.WithLogger(c.logical, func() ([]byte, error) {
		time.Sleep(c.delay)

		b, err := fun()
		if err != nil {
			c.Connection.Close()
		}
		return b, err
	})
}

func (c *Connection) ReadCoils(address, quantity uint16) ([]byte, error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().ReadCoils(address, quantity)
	})
}

func (c *Connection) WriteSingleCoil(address, value uint16) ([]byte, error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().WriteSingleCoil(address, value)
	})
}

func (c *Connection) ReadInputRegisters(address, quantity uint16) ([]byte, error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().ReadInputRegisters(address, quantity)
	})
}

func (c *Connection) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().ReadHoldingRegisters(address, quantity)
	})
}

func (c *Connection) WriteSingleRegister(address, value uint16) ([]byte, error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().WriteSingleRegister(address, value)
	})
}

func (c *Connection) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().WriteMultipleRegisters(address, quantity, value)
	})
}

func (c *Connection) ReadDiscreteInputs(address, quantity uint16) (results []byte, err error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().ReadDiscreteInputs(address, quantity)
	})
}

func (c *Connection) WriteMultipleCoils(address, quantity uint16, value []byte) (results []byte, err error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().WriteMultipleCoils(address, quantity, value)
	})
}

func (c *Connection) ReadWriteMultipleRegisters(readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) (results []byte, err error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().ReadWriteMultipleRegisters(readAddress, readQuantity, writeAddress, writeQuantity, value)
	})
}

func (c *Connection) MaskWriteRegister(address, andMask, orMask uint16) (results []byte, err error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().MaskWriteRegister(address, andMask, orMask)
	})
}

func (c *Connection) ReadFIFOQueue(address uint16) (results []byte, err error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().ReadFIFOQueue(address)
	})
}
