package provider

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	gridx "github.com/grid-x/modbus"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// ModbusMbmd implements modbus RTU and TCP access
type ModbusMbmd struct {
	log   *util.Logger
	conn  *modbus.Connection
	op    rs485.Operation
	scale float64
}

func init() {
	registry.Add("mbmd", NewModbusMbmdFromConfig)
}

// NewModbusMbmdFromConfig creates Modbus plugin
func NewModbusMbmdFromConfig(other map[string]interface{}) (Provider, error) {
	cc := struct {
		Model           string
		modbus.Settings `mapstructure:",squash"`
		Value           string
		Scale           float64
		Delay           time.Duration
		ConnectDelay    time.Duration
		Timeout         time.Duration
	}{
		Scale: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// assume RTU if not set and this is a known RS485 meter model
	if cc.RTU == nil {
		b := modbus.IsRS485(cc.Model)
		cc.RTU = &b
	}

	conn, err := modbus.NewConnection(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID)
	if err != nil {
		return nil, err
	}

	// set non-default timeout
	if cc.Timeout > 0 {
		conn.Timeout(cc.Timeout)
	}

	// set non-default delay
	if cc.Delay > 0 {
		conn.Delay(cc.Delay)
	}

	// set non-default connect delay
	if cc.ConnectDelay > 0 {
		conn.ConnectDelay(cc.ConnectDelay)
	}

	log := util.NewLogger("mbmd")
	conn.Logger(log.TRACE)

	if cc.Value == "" {
		return nil, errors.New("value is required")
	}

	if cc.Model == "" {
		return nil, errors.New("model ist required")
	}

	device, err := modbus.NewDevice(cc.Model, 0)
	if err != nil {
		return nil, err
	}

	measurement, err := meters.MeasurementString(cc.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %s", cc.Value)
	}

	device, ok := device.(*rs485.RS485)
	if !ok {
		return nil, fmt.Errorf("invalid device: %s", cc.Model)
	}

	op, err := modbus.RS485FindDeviceOp(device.(*rs485.RS485), measurement)
	if err != nil {
		return nil, fmt.Errorf("invalid value for device: %s", cc.Value)
	}

	mb := &ModbusMbmd{
		log:   log,
		conn:  conn,
		op:    op,
		scale: cc.Scale,
	}
	return mb, nil
}

func (m *ModbusMbmd) bytesGetter() ([]byte, error) {
	switch m.op.FuncCode {
	case gridx.FuncCodeReadHoldingRegisters:
		return m.conn.ReadHoldingRegisters(m.op.OpCode, m.op.ReadLen)

	case gridx.FuncCodeReadInputRegisters:
		return m.conn.ReadInputRegisters(m.op.OpCode, m.op.ReadLen)

	case gridx.FuncCodeReadCoils:
		return m.conn.ReadCoils(m.op.OpCode, m.op.ReadLen)

	default:
		return nil, fmt.Errorf("invalid read function code: %d", m.op.FuncCode)
	}
}

func (m *ModbusMbmd) floatGetter() (f float64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	var bytes []byte
	if bytes, err = m.bytesGetter(); err != nil {
		return 0, fmt.Errorf("read failed: %w", err)
	}

	return m.scale * m.op.Transform(bytes), nil
}

var _ FloatProvider = (*Modbus)(nil)

// FloatGetter executes configured modbus read operation and implements func() (float64, error)
func (m *ModbusMbmd) FloatGetter() (func() (f float64, err error), error) {
	return m.floatGetter, nil
}

var _ IntProvider = (*Modbus)(nil)

// IntGetter executes configured modbus read operation and implements IntProvider
func (m *ModbusMbmd) IntGetter() (func() (int64, error), error) {
	g, err := m.FloatGetter()

	return func() (int64, error) {
		res, err := g()
		return int64(math.Round(res)), err
	}, err
}

var _ StringProvider = (*Modbus)(nil)

// StringGetter executes configured modbus read operation and implements IntProvider
func (m *ModbusMbmd) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		b, err := m.bytesGetter()
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(bytes.TrimLeft(b, "\x00"))), nil
	}, nil
}

var _ BoolProvider = (*Modbus)(nil)

// BoolGetter executes configured modbus read operation and implements IntProvider
func (m *ModbusMbmd) BoolGetter() (func() (bool, error), error) {
	return func() (bool, error) {
		bytes, err := m.bytesGetter()
		if err != nil {
			return false, err
		}

		u, err := UintFromBytes(bytes)
		return u > 0, err
	}, nil
}
