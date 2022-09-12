package provider

import (
	"encoding/binary"
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
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

// Modbus implements modbus RTU and TCP access
type Modbus struct {
	log    *util.Logger
	conn   *modbus.Connection
	device meters.Device
	op     modbus.Operation
	scale  float64
}

func init() {
	registry.Add("modbus", NewModbusFromConfig)
}

// NewModbusFromConfig creates Modbus plugin
func NewModbusFromConfig(other map[string]interface{}) (IntProvider, error) {
	cc := struct {
		Model           string
		modbus.Settings `mapstructure:",squash"`
		Register        modbus.Register
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

	log := util.NewLogger("modbus")
	conn.Logger(log.TRACE)

	var device meters.Device
	var op modbus.Operation

	if cc.Value != "" && cc.Register.Decode != "" {
		return nil, errors.New("modbus cannot have value and register both")
	}

	if cc.Value == "" && cc.Register.Decode == "" {
		log.WARN.Println("missing modbus value or register - assuming Power")
		cc.Value = "Power"
	}

	if cc.Model == "" && cc.Value != "" {
		return nil, errors.New("need device model when value configured")
	}

	// no registered configured - need device
	if cc.Register.Decode == "" {
		device, err = modbus.NewDevice(cc.Model, cc.SubDevice)

		// prepare device
		if err == nil {
			err = device.Initialize(conn)

			// silence KOSTAL implementation errors
			if errors.Is(err, meters.ErrPartiallyOpened) {
				err = nil
			}
		}

		if err != nil {
			return nil, err
		}
	}

	// model + value configured
	if cc.Value != "" {
		cc.Value = modbus.ReadingName(cc.Value)
		if err := modbus.ParseOperation(device, cc.Value, &op); err != nil {
			return nil, fmt.Errorf("invalid value %s", cc.Value)
		}
	}

	// register configured
	if cc.Register.Decode != "" {
		if op.MBMD, err = modbus.RegisterOperation(cc.Register); err != nil {
			return nil, err
		}
	}

	mb := &Modbus{
		log:    log,
		conn:   conn,
		device: device,
		op:     op,
		scale:  cc.Scale,
	}
	return mb, nil
}

func (m *Modbus) bytesGetter() ([]byte, error) {
	if op := m.op.MBMD; op.FuncCode != 0 {
		switch op.FuncCode {
		case rs485.ReadHoldingReg:
			return m.conn.ReadHoldingRegisters(op.OpCode, op.ReadLen)
		case rs485.ReadInputReg:
			return m.conn.ReadInputRegisters(op.OpCode, op.ReadLen)
		default:
			return nil, fmt.Errorf("unknown function code %d", op.FuncCode)
		}
	}

	return nil, errors.New("expected rtu reading")
}

func (m *Modbus) floatGetter() (f float64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	var res meters.MeasurementResult

	// if funccode is configured, execute the read directly
	if op := m.op.MBMD; op.FuncCode != 0 {
		var bytes []byte
		if bytes, err = m.bytesGetter(); err != nil {
			return 0, fmt.Errorf("read failed: %w", err)
		}

		return m.scale * op.Transform(bytes), nil
	}

	// if funccode is not configured, try find the reading on sunspec
	if dev, ok := m.device.(*sunspec.SunSpec); ok {
		if m.op.MBMD.IEC61850 != 0 {
			// client := m.conn.ModbusClient()
			res, err = dev.QueryOp(m.conn, m.op.MBMD.IEC61850)
		} else {
			if res.Value, err = dev.QueryPoint(
				m.conn,
				m.op.SunSpec.Model,
				m.op.SunSpec.Block,
				m.op.SunSpec.Point,
			); err != nil {
				err = fmt.Errorf("model %d block %d point %s: %w", m.op.SunSpec.Model, m.op.SunSpec.Block, m.op.SunSpec.Point, err)
			}
		}
	}

	// silence NaN reading errors by assuming zero
	if err != nil && errors.Is(err, meters.ErrNaN) {
		res.Value = 0
		err = nil
	}

	if err == nil {
		if m.op.MBMD.IEC61850 != 0 {
			m.log.TRACE.Printf("%s: %v", m.op.MBMD.IEC61850, res.Value)
		} else {
			m.log.TRACE.Printf("%d:%d:%s: %v", m.op.SunSpec.Model, m.op.SunSpec.Block, m.op.SunSpec.Point, res.Value)
		}
	}

	return m.scale * res.Value, err
}

// FloatGetter executes configured modbus read operation and implements func() (float64, error)
func (m *Modbus) FloatGetter() func() (f float64, err error) {
	return m.floatGetter
}

// IntGetter executes configured modbus read operation and implements IntProvider
func (m *Modbus) IntGetter() func() (int64, error) {
	g := m.FloatGetter()

	return func() (int64, error) {
		res, err := g()
		return int64(math.Round(res)), err
	}
}

// StringGetter executes configured modbus read operation and implements IntProvider
func (m *Modbus) StringGetter() func() (string, error) {
	return func() (string, error) {
		bytes, err := m.bytesGetter()
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(bytes)), nil
	}
}

// UintFromBytes converts byte slice to bigendian uint value
func UintFromBytes(bytes []byte) (u uint64, err error) {
	switch l := len(bytes); l {
	case 2:
		u = uint64(binary.BigEndian.Uint16(bytes))
	case 4:
		u = uint64(binary.BigEndian.Uint32(bytes))
	case 8:
		u = binary.BigEndian.Uint64(bytes)
	default:
		err = fmt.Errorf("unexpected length: %d", l)
	}

	return u, err
}

// BoolGetter executes configured modbus read operation and implements IntProvider
func (m *Modbus) BoolGetter() func() (bool, error) {
	return func() (bool, error) {
		bytes, err := m.bytesGetter()
		if err != nil {
			return false, err
		}

		u, err := UintFromBytes(bytes)
		return u > 0, err
	}
}

// IntSetter executes configured modbus write operation and implements SetIntProvider
func (m *Modbus) IntSetter(param string) func(int64) error {
	return func(val int64) error {
		var err error

		// if funccode is configured, execute the read directly
		if op := m.op.MBMD; op.FuncCode != 0 {
			uval := uint16(int64(m.scale) * val)

			switch op.FuncCode {
			case gridx.FuncCodeWriteSingleRegister:
				_, err = m.conn.WriteSingleRegister(op.OpCode, uval)
			default:
				err = fmt.Errorf("unknown function code %d", op.FuncCode)
			}
		} else {
			err = errors.New("modbus plugin does not support writing to sunspec")
		}

		return err
	}
}

// BoolSetter executes configured modbus write operation and implements SetBoolProvider
func (m *Modbus) BoolSetter(param string) func(bool) error {
	set := m.IntSetter(param)

	return func(val bool) error {
		var ival int64
		if val {
			ival = 1
		}

		return set(ival)
	}
}
