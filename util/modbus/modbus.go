package modbus

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

type Protocol int

const (
	Tcp Protocol = iota
	Rtu
	Ascii

	CoilOn uint16 = 0xFF00
)

// Settings contains the ModBus TCP settings
// RTU field is included for compatibility with modbus.tpl which renders rtu: false for TCP
// TODO remove RTU field (https://github.com/evcc-io/evcc/issues/3360)
type TcpSettings struct {
	URI string
	ID  uint8
	RTU *bool `mapstructure:"rtu"`
}

// Settings contains the ModBus settings
type Settings struct {
	ID                  uint8
	SubDevice           int
	URI, Device, Comset string
	Baudrate            int
	RTU                 *bool // indicates RTU over TCP if true
}

func (s *Settings) String() string {
	if s.URI != "" {
		return s.URI
	}
	return s.Device
}

// Connection decorates a meters.Connection with transparent slave id and error handling
type Connection struct {
	slaveID uint8
	mu      sync.Mutex
	conn    meters.Connection
	delay   time.Duration
}

func (mb *Connection) prepare(slaveID uint8) {
	mb.conn.Slave(slaveID)
	if mb.delay > 0 {
		time.Sleep(mb.delay)
	}
}

func (mb *Connection) handle(res []byte, err error) ([]byte, error) {
	if err != nil {
		mb.conn.Close()
	}
	return res, err
}

// Delay sets delay so use between subsequent modbus operations
func (mb *Connection) Delay(delay time.Duration) {
	mb.delay = delay
}

// ConnectDelay sets the initial delay after connecting before starting communication
func (mb *Connection) ConnectDelay(delay time.Duration) {
	mb.conn.ConnectDelay(delay)
}

// Logger sets logger implementation
func (mb *Connection) Logger(logger meters.Logger) {
	mb.conn.Logger(logger)
}

// Timeout sets the connection timeout (not idle timeout)
func (mb *Connection) Timeout(timeout time.Duration) {
	mb.conn.Timeout(timeout)
}

// ReadCoils wraps the underlying implementation
func (mb *Connection) ReadCoilsWithSlave(slaveID uint8, address, quantity uint16) ([]byte, error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.prepare(slaveID)
	return mb.handle(mb.conn.ModbusClient().ReadCoils(address, quantity))
}

// WriteSingleCoil wraps the underlying implementation
func (mb *Connection) WriteSingleCoilWithSlave(slaveID uint8, address, value uint16) ([]byte, error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.prepare(slaveID)
	return mb.handle(mb.conn.ModbusClient().WriteSingleCoil(address, value))
}

// ReadInputRegisters wraps the underlying implementation
func (mb *Connection) ReadInputRegistersWithSlave(slaveID uint8, address, quantity uint16) ([]byte, error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.prepare(slaveID)
	return mb.handle(mb.conn.ModbusClient().ReadInputRegisters(address, quantity))
}

// ReadHoldingRegisters wraps the underlying implementation
func (mb *Connection) ReadHoldingRegistersWithSlave(slaveID uint8, address, quantity uint16) ([]byte, error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.prepare(slaveID)
	return mb.handle(mb.conn.ModbusClient().ReadHoldingRegisters(address, quantity))
}

// WriteSingleRegister wraps the underlying implementation
func (mb *Connection) WriteSingleRegisterWithSlave(slaveID uint8, address, value uint16) ([]byte, error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.prepare(slaveID)
	return mb.handle(mb.conn.ModbusClient().WriteSingleRegister(address, value))
}

// WriteMultipleRegisters wraps the underlying implementation
func (mb *Connection) WriteMultipleRegistersWithSlave(slaveID uint8, address, quantity uint16, value []byte) ([]byte, error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.prepare(slaveID)
	return mb.handle(mb.conn.ModbusClient().WriteMultipleRegisters(address, quantity, value))
}

// ReadDiscreteInputs wraps the underlying implementation
func (mb *Connection) ReadDiscreteInputsWithSlave(slaveID uint8, address, quantity uint16) (results []byte, err error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.prepare(slaveID)
	return mb.handle(mb.conn.ModbusClient().ReadDiscreteInputs(address, quantity))
}

// WriteMultipleCoils wraps the underlying implementation
func (mb *Connection) WriteMultipleCoilsWithSlave(slaveID uint8, address, quantity uint16, value []byte) (results []byte, err error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.prepare(slaveID)
	return mb.handle(mb.conn.ModbusClient().WriteMultipleCoils(address, quantity, value))
}

// ReadWriteMultipleRegisters wraps the underlying implementation
func (mb *Connection) ReadWriteMultipleRegistersWithSlave(slaveID uint8, readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) (results []byte, err error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.prepare(slaveID)
	return mb.handle(mb.conn.ModbusClient().ReadWriteMultipleRegisters(readAddress, readQuantity, writeAddress, writeQuantity, value))
}

// MaskWriteRegister wraps the underlying implementation
func (mb *Connection) MaskWriteRegisterWithSlave(slaveID uint8, address, andMask, orMask uint16) (results []byte, err error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.prepare(slaveID)
	return mb.handle(mb.conn.ModbusClient().MaskWriteRegister(address, andMask, orMask))
}

// ReadFIFOQueue wraps the underlying implementation
func (mb *Connection) ReadFIFOQueueWithSlave(slaveID uint8, address uint16) (results []byte, err error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.prepare(slaveID)
	return mb.handle(mb.conn.ModbusClient().ReadFIFOQueue(address))
}

func (mb *Connection) ReadCoils(address, quantity uint16) ([]byte, error) {
	return mb.ReadCoilsWithSlave(mb.slaveID, address, quantity)
}

func (mb *Connection) WriteSingleCoil(address, value uint16) ([]byte, error) {
	return mb.WriteSingleCoilWithSlave(mb.slaveID, address, value)
}

func (mb *Connection) ReadInputRegisters(address, quantity uint16) ([]byte, error) {
	return mb.ReadInputRegistersWithSlave(mb.slaveID, address, quantity)
}

func (mb *Connection) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	return mb.ReadHoldingRegistersWithSlave(mb.slaveID, address, quantity)
}

func (mb *Connection) WriteSingleRegister(address, value uint16) ([]byte, error) {
	return mb.WriteSingleRegisterWithSlave(mb.slaveID, address, value)
}

func (mb *Connection) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	return mb.WriteMultipleRegistersWithSlave(mb.slaveID, address, quantity, value)
}

func (mb *Connection) ReadDiscreteInputs(address, quantity uint16) (results []byte, err error) {
	return mb.ReadDiscreteInputsWithSlave(mb.slaveID, address, quantity)
}

func (mb *Connection) WriteMultipleCoils(address, quantity uint16, value []byte) (results []byte, err error) {
	return mb.WriteMultipleCoilsWithSlave(mb.slaveID, address, quantity, value)
}

func (mb *Connection) ReadWriteMultipleRegisters(readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) (results []byte, err error) {
	return mb.ReadWriteMultipleRegistersWithSlave(mb.slaveID, readAddress, readQuantity, writeAddress, writeQuantity, value)
}

func (mb *Connection) MaskWriteRegister(address, andMask, orMask uint16) (results []byte, err error) {
	return mb.MaskWriteRegisterWithSlave(mb.slaveID, address, andMask, orMask)
}

func (mb *Connection) ReadFIFOQueue(address uint16) (results []byte, err error) {
	return mb.ReadFIFOQueueWithSlave(mb.slaveID, address)
}

var (
	connections = make(map[string]meters.Connection)
	mu          sync.Mutex
)

func registeredConnection(key string, newConn meters.Connection) meters.Connection {
	mu.Lock()
	defer mu.Unlock()

	if conn, ok := connections[key]; ok {
		return conn
	}

	connections[key] = newConn

	return newConn
}

// ProtocolFromRTU identifies the wire format from the RTU setting
func ProtocolFromRTU(rtu *bool) Protocol {
	if rtu != nil && *rtu {
		return Rtu
	}
	return Tcp
}

// NewConnection creates physical modbus device from config
func NewConnection(uri, device, comset string, baudrate int, proto Protocol, slaveID uint8) (*Connection, error) {
	var conn meters.Connection

	if device != "" && uri != "" {
		return nil, errors.New("invalid modbus configuration: can only have either uri or device")
	}

	if device != "" {
		switch strings.ToUpper(comset) {
		case "8N1", "8E1", "8N2":
		case "80":
			comset = "8E1"
		default:
			return nil, fmt.Errorf("invalid comset: %s", comset)
		}

		if baudrate == 0 {
			return nil, errors.New("invalid modbus configuration: need baudrate and comset")
		}

		if proto == Ascii {
			conn = registeredConnection(device, meters.NewASCII(device, baudrate, comset))
		} else {
			conn = registeredConnection(device, meters.NewRTU(device, baudrate, comset))
		}
	}

	if uri != "" {
		uri = util.DefaultPort(uri, 502)

		switch proto {
		case Rtu:
			conn = registeredConnection(uri, meters.NewRTUOverTCP(uri))
		case Ascii:
			conn = registeredConnection(uri, meters.NewASCIIOverTCP(uri))
		default:
			conn = registeredConnection(uri, meters.NewTCP(uri))
		}
	}

	if conn == nil {
		return nil, errors.New("invalid modbus configuration: need either uri or device")
	}

	slaveConn := &Connection{
		slaveID: slaveID,
		conn:    conn,
	}

	return slaveConn, nil
}

// NewDevice creates physical modbus device from config
func NewDevice(model string, subdevice int) (device meters.Device, err error) {
	if IsRS485(model) {
		device, err = rs485.NewDevice(strings.ToUpper(model))
	} else {
		device = sunspec.NewDevice(strings.ToUpper(model), subdevice)
	}

	if device == nil {
		err = errors.New("invalid modbus configuration: need either uri or device")
	}

	return device, err
}

// IsRS485 determines if model is a known MBMD rs485 device model
func IsRS485(model string) bool {
	for k := range rs485.Producers {
		if strings.EqualFold(model, k) {
			return true
		}
	}
	return false
}

// RS485FindDeviceOp checks is RS485 device supports operation
func RS485FindDeviceOp(device *rs485.RS485, measurement meters.Measurement) (op rs485.Operation, err error) {
	ops := device.Producer().Produce()

	for _, op := range ops {
		if op.IEC61850 == measurement {
			return op, nil
		}
	}

	return op, fmt.Errorf("unsupported measurement: %s", measurement.String())
}
