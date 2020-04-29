package modbus

import (
	"strings"

	"github.com/andig/evcc/util"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

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
