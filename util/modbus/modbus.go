package modbus

import (
	"errors"
	"fmt"
	"strings"
	"sync"

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
	Udp

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
	UDP                 bool
	RTU                 *bool // indicates RTU over TCP if true
}

// Protocol identifies the wire format from the RTU setting
func (s Settings) Protocol() Protocol {
	switch {
	case s.UDP:
		return Udp
	case s.RTU != nil && *s.RTU:
		return Rtu
	default:
		return Tcp
	}
}

func (s *Settings) String() string {
	if s.URI != "" {
		return s.URI
	}
	return s.Device
}

type meterConnection struct {
	meters.Connection
	proto Protocol
	*logger
}

var (
	connections = make(map[string]*meterConnection)
	mu          sync.Mutex
)

func registeredConnection(key string, proto Protocol, newConn meters.Connection) (*meterConnection, error) {
	mu.Lock()
	defer mu.Unlock()

	if conn, ok := connections[key]; ok {
		if conn.proto != proto {
			return nil, fmt.Errorf("connection already registered with different protocol: %s", key)
		}

		return conn, nil
	}

	connection := &meterConnection{
		Connection: newConn,
		proto:      proto,
		logger:     new(logger),
	}

	newConn.Logger(connection.logger)
	connections[key] = connection

	return connection, nil
}

// NewConnection creates physical modbus device from config
func NewConnection(uri, device, comset string, baudrate int, proto Protocol, slaveID uint8) (*Connection, error) {
	conn, err := physicalConnection(proto, Settings{
		URI:      uri,
		Device:   device,
		Comset:   comset,
		Baudrate: baudrate,
	})
	if err != nil {
		return nil, err
	}

	res := &Connection{
		Connection: conn.Clone(slaveID),
		logger:     conn.logger,
	}

	return res, nil
}

func physicalConnection(proto Protocol, cfg Settings) (*meterConnection, error) {
	if (cfg.Device != "") == (cfg.URI != "") {
		return nil, errors.New("invalid modbus configuration: must have either uri or device")
	}

	if cfg.Device != "" {
		switch strings.ToUpper(cfg.Comset) {
		case "8N1", "8E1", "8N2":
		case "80":
			cfg.Comset = "8E1"
		default:
			return nil, fmt.Errorf("invalid comset: %s", cfg.Comset)
		}

		if cfg.Baudrate == 0 {
			return nil, errors.New("invalid modbus configuration: need baudrate and comset")
		}

		if proto == Ascii {
			return registeredConnection(cfg.Device, Ascii, meters.NewASCII(cfg.Device, cfg.Baudrate, cfg.Comset))
		} else {
			return registeredConnection(cfg.Device, Rtu, meters.NewRTU(cfg.Device, cfg.Baudrate, cfg.Comset))
		}
	}

	uri := util.DefaultPort(cfg.URI, 502)

	switch proto {
	case Udp:
		return registeredConnection(uri, Udp, meters.NewRTUOverUDP(uri))
	case Rtu:
		return registeredConnection(uri, Rtu, meters.NewRTUOverTCP(uri))
	case Ascii:
		return registeredConnection(uri, Ascii, meters.NewASCIIOverTCP(uri))
	default:
		return registeredConnection(uri, Tcp, meters.NewTCP(uri))
	}
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
