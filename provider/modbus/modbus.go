package modbus

import (
	"fmt"
	"strings"

	"github.com/andig/evcc/util"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

// Register contains the ModBus register configuration
type Register struct {
	Address, Length uint16
	Type            string
	Decode          string
}

// RegisterOperation creates a read operation from a register definition
func RegisterOperation(r Register) (rs485.Operation, error) {
	op := rs485.Operation{
		OpCode:  r.Address,
		ReadLen: r.Length,
	}

	switch strings.ToLower(r.Type) {
	case "holding":
		op.FuncCode = rs485.ReadHoldingReg
	case "input":
		op.FuncCode = rs485.ReadInputReg
	default:
		return rs485.Operation{}, fmt.Errorf("invalid register type: %s", r.Type)
	}

	switch strings.ToLower(r.Decode) {
	case "float32", "ieee754":
		op.Transform = rs485.RTUIeee754ToFloat64
	case "float64":
		op.Transform = rs485.RTUUint64ToFloat64
	case "uint16":
		op.Transform = rs485.RTUUint16ToFloat64
	case "uint32":
		op.Transform = rs485.RTUUint32ToFloat64
	case "uint32s":
		op.Transform = rs485.RTUUint32ToFloat64Swapped
	case "uint64":
		op.Transform = rs485.RTUUint64ToFloat64
	case "int16":
		op.Transform = rs485.RTUInt16ToFloat64
	case "int32":
		op.Transform = rs485.RTUInt32ToFloat64
	case "int32s":
		op.Transform = rs485.RTUInt32ToFloat64Swapped
	default:
		return rs485.Operation{}, fmt.Errorf("invalid register decoding: %s", r.Decode)
	}

	return op, nil
}

// Connection contains the ModBus connection configuration
type Connection struct {
	ID                  uint8
	URI, Device, Comset string
	Baudrate            int
	RTU                 *bool // indicates RTU over TCP if true
}

var connections map[string]meters.Connection

func registeredConnection(key string, newConn meters.Connection) meters.Connection {
	if connections == nil {
		connections = make(map[string]meters.Connection)
	}

	if conn, ok := connections[key]; ok {
		return conn
	}

	connections[key] = newConn
	return newConn
}

// NewConnection creates physical modbus device from config
func NewConnection(log *util.Logger, uri, device, comset string, baudrate int, rtu bool) (conn meters.Connection) {
	if device != "" {
		conn = registeredConnection(device, meters.NewRTU(device, baudrate, comset))
	}

	if uri != "" {
		if rtu {
			conn = registeredConnection(uri, meters.NewRTUOverTCP(uri))
		} else {
			conn = registeredConnection(uri, meters.NewTCP(uri))
		}
	}

	if conn == nil {
		log.FATAL.Fatalf("config: invalid modbus configuration: need either uri or device")
	}

	return conn
}

// NewDevice creates physical modbus device from config
func NewDevice(log *util.Logger, model string, isRS485 bool) (device meters.Device, err error) {
	if isRS485 {
		device, err = rs485.NewDevice(strings.ToUpper(model))
	} else {
		device = sunspec.NewDevice(strings.ToUpper(model))
	}

	if device == nil {
		log.FATAL.Fatalf("config: invalid modbus configuration: need either uri or device")
	}

	return device, err
}

// IsRS485 determines if model is a known MBMD rs485 device model
func IsRS485(model string) bool {
	for k := range rs485.Producers {
		if strings.ToUpper(model) == k {
			return true
		}
	}
	return false
}

// RS485FindDeviceOp checks is RS485 device supports operation
func RS485FindDeviceOp(device *rs485.RS485, measurement meters.Measurement) (op rs485.Operation) {
	ops := device.Producer().Produce()

	for _, o := range ops {
		if o.IEC61850 == measurement {
			op = o
			break
		}
	}

	return op
}
