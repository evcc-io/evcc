package modbus

import (
	"errors"
	"fmt"
	"io"
	"syscall"
	"time"

	gridx "github.com/grid-x/modbus"
	"github.com/volkszaehler/mbmd/meters"
)

// Connection is a logical modbus connection per slave ID sharing a physical connection
type Connection struct {
	*logger
	meters.Connection
	physical *meterConnection
	slaveID  uint8 // duplicated from meters.Connection
	logical  meters.Logger
}

func (c *Connection) Addr() string {
	return fmt.Sprintf("%s::%d", c.Connection.String(), c.slaveID)
}

func (c *Connection) Logger(logger meters.Logger) {
	c.logical = logger
}

// Delay applies the delay to the shared physical connection
func (c *Connection) Delay(delay time.Duration) {
	c.physical.setDelay(delay)
}

func (c *Connection) Clone(slaveID uint8) *Connection {
	return &Connection{
		slaveID:    slaveID,
		Connection: c.Connection.Clone(slaveID),
		logger:     c.logger,
		physical:   c.physical,
	}
}

// ConnectDelay applies the connect delay to the shared physical connection
func (c *Connection) ConnectDelay(delay time.Duration) {
	c.physical.setConnectDelay(delay)
}

// Timeout applies the timeout to the shared physical connection
func (c *Connection) Timeout(timeout time.Duration) {
	c.physical.setTimeout(timeout)
}

// reconnectable reports errors where the peer dropped the socket: the request
// never reached the device (EPIPE, ECONNRESET) or got no reply (EOF).
func reconnectable(err error) bool {
	return errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF)
}

func (c *Connection) exec(fun func() ([]byte, error)) ([]byte, error) {
	return c.WithLogger(c.logical, func() ([]byte, error) {
		time.Sleep(c.physical.getDelay())

		b, err := fun()
		if err != nil {
			c.Connection.Close()
		}
		return b, err
	})
}

// read runs an idempotent read and reissues it once on a fresh connection after
// the peer closed the idle one. Writes stay on exec: they may already have executed.
func (c *Connection) read(fun func() ([]byte, error)) ([]byte, error) {
	b, err := c.exec(fun)
	if err != nil && reconnectable(err) {
		b, err = c.exec(fun)
	}
	return b, err
}

func (c *Connection) ReadCoils(address, quantity uint16) ([]byte, error) {
	return c.read(func() ([]byte, error) {
		return c.ModbusClient().ReadCoils(address, quantity)
	})
}

func (c *Connection) WriteSingleCoil(address, value uint16) ([]byte, error) {
	return c.exec(func() ([]byte, error) {
		return c.ModbusClient().WriteSingleCoil(address, value)
	})
}

func (c *Connection) ReadInputRegisters(address, quantity uint16) ([]byte, error) {
	return c.read(func() ([]byte, error) {
		return c.ModbusClient().ReadInputRegisters(address, quantity)
	})
}

func (c *Connection) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	return c.read(func() ([]byte, error) {
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
	return c.read(func() ([]byte, error) {
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
	return c.read(func() ([]byte, error) {
		return c.ModbusClient().ReadFIFOQueue(address)
	})
}

// maximum PDU size (253 bytes) minus the function code
const maxRawDataLength = 252

// ErrRawUnsupported is returned by SendRaw for connections other than Modbus TCP
var ErrRawUnsupported = errors.New("raw requests require a modbus tcp connection")

// SendRaw sends a Modbus TCP PDU without retries and returns the response data.
// Exception responses are returned as *modbus.Error and keep the connection open.
func (c *Connection) SendRaw(functionCode byte, data []byte) ([]byte, error) {
	tcp, ok := c.Connection.(*meters.TCP)
	if !ok {
		return nil, ErrRawUnsupported
	}

	if len(data) > maxRawDataLength {
		return nil, fmt.Errorf("raw request data exceeds %d bytes", maxRawDataLength)
	}

	// exec closes the connection on error, so exceptions are returned afterwards
	var exception error

	b, err := c.exec(func() ([]byte, error) {
		req, err := tcp.Handler.Encode(&gridx.ProtocolDataUnit{FunctionCode: functionCode, Data: data})
		if err != nil {
			return nil, err
		}

		res, err := tcp.Handler.Send(req)
		if err != nil {
			return nil, err
		}

		if err := tcp.Handler.Verify(req, res); err != nil {
			return nil, err
		}

		pdu, err := tcp.Handler.Decode(res)
		if err != nil {
			return nil, err
		}

		switch {
		case pdu.FunctionCode == functionCode:
			return pdu.Data, nil
		case pdu.FunctionCode == 0x80|functionCode && len(pdu.Data) == 1:
			exception = &gridx.Error{FunctionCode: pdu.FunctionCode, ExceptionCode: pdu.Data[0]}
			return nil, nil
		default:
			return nil, fmt.Errorf("invalid raw response: function code %02x, data % x", pdu.FunctionCode, pdu.Data)
		}
	})

	if err == nil {
		err = exception
	}

	return b, err
}
