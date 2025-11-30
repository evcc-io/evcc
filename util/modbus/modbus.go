package modbus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/volkszaehler/mbmd/meters"
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
	ID        uint8  `json:"id,omitempty" yaml:",omitempty"`
	SubDevice int    `json:"subdevice,omitempty" yaml:",omitempty"`
	URI       string `json:"uri,omitempty" yaml:",omitempty"`
	Device    string `json:"device,omitempty" yaml:",omitempty"`
	Comset    string `json:"comset,omitempty" yaml:",omitempty"`
	Baudrate  int    `json:"baudrate,omitempty" yaml:",omitempty"`
	UDP       bool   `json:"udp,omitempty" yaml:",omitempty"`
	RTU       *bool  `json:"rtu,omitempty" yaml:",omitempty"`
}

// Protocol identifies the wire format from the RTU setting
func (s Settings) Protocol() Protocol {
	switch {
	case s.UDP:
		return Udp
	case s.Device != "" || s.RTU != nil && *s.RTU:
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
	refs  int // count of references; first connection has ref count 0
	*logger
}

var (
	connections = make(map[string]*meterConnection)
	mu          sync.Mutex
)

func unregisterConnection(key string) {
	mu.Lock()
	defer mu.Unlock()

	conn, ok := connections[key]
	if !ok {
		panic("unregisterConnection: connection not found " + key)
	}

	if conn.refs > 0 {
		conn.refs--
		return
	}

	delete(connections, key)
}

func registeredConnection(ctx context.Context, key string, proto Protocol, newConn meters.Connection) (*meterConnection, error) {
	mu.Lock()
	defer mu.Unlock()

	if conn, ok := connections[key]; ok {
		if conn.proto != proto {
			return nil, fmt.Errorf("connection already registered with different protocol: %s", key)
		}

		conn.refs++

		return conn, nil
	}

	go func() {
		<-ctx.Done()
		unregisterConnection(key)
	}()

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
func NewConnection(ctx context.Context, uri, device, comset string, baudrate int, proto Protocol, slaveID uint8) (*Connection, error) {
	conn, err := physicalConnection(ctx, proto, Settings{
		URI:      uri,
		Device:   device,
		Comset:   comset,
		Baudrate: baudrate,
	})
	if err != nil {
		return nil, err
	}

	res := &Connection{
		slaveID:    slaveID,
		Connection: conn.Clone(slaveID),
		logger:     conn.logger,
	}

	return res, nil
}

func physicalConnection(ctx context.Context, proto Protocol, cfg Settings) (*meterConnection, error) {
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

		switch proto {
		case Ascii:
			return registeredConnection(ctx, cfg.Device, proto, meters.NewASCII(cfg.Device, cfg.Baudrate, cfg.Comset))
		default:
			return registeredConnection(ctx, cfg.Device, proto, meters.NewRTU(cfg.Device, cfg.Baudrate, cfg.Comset))
		}
	}

	uri := util.DefaultPort(cfg.URI, 502)

	switch proto {
	case Udp:
		return registeredConnection(ctx, uri, proto, meters.NewRTUOverUDP(uri))

	case Rtu:
		// use retry outside of grid-x/modbus
		conn := meters.NewRTUOverTCP(uri)
		conn.Handler.LinkRecoveryTimeout = 0
		conn.Handler.ProtocolRecoveryTimeout = 0

		return registeredConnection(ctx, uri, proto, conn)

	case Ascii:
		// use retry outside of grid-x/modbus
		conn := meters.NewASCIIOverTCP(uri)
		conn.Handler.LinkRecoveryTimeout = 0
		conn.Handler.ProtocolRecoveryTimeout = 0

		return registeredConnection(ctx, uri, proto, conn)

	default:
		// use retry outside of grid-x/modbus
		conn := meters.NewTCP(uri)
		conn.Handler.LinkRecoveryTimeout = 0
		conn.Handler.ProtocolRecoveryTimeout = 0

		return registeredConnection(ctx, uri, proto, conn)
	}
}
