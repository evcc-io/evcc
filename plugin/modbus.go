package plugin

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	gridx "github.com/grid-x/modbus"
)

// Modbus implements modbus RTU and TCP access
type Modbus struct {
	log   *util.Logger
	conn  *modbus.Connection
	reg   modbus.Register
	scale float64
}

func init() {
	registry.AddCtx("modbus", NewModbusFromConfig)
}

// NewModbusFromConfig creates Modbus plugin
func NewModbusFromConfig(ctx context.Context, other map[string]interface{}) (Plugin, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		Register        modbus.Register
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

	modbus.Lock()
	defer modbus.Unlock()

	conn, err := modbus.NewConnection(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Settings.Protocol(), cc.ID)
	if err != nil {
		return nil, err
	}

	// set non-default timeout
	conn.Timeout(cc.Timeout)

	// set non-default delay
	conn.Delay(cc.Delay)

	// set non-default connect delay
	conn.ConnectDelay(cc.ConnectDelay)

	log := contextLogger(ctx, util.NewLogger("modbus"))
	conn.Logger(log.TRACE)

	if err := cc.Register.Error(); err != nil {
		return nil, err
	}

	mb := &Modbus{
		log:   log,
		conn:  conn,
		reg:   cc.Register,
		scale: cc.Scale,
	}
	return mb, nil
}

func (m *Modbus) readBytes(op modbus.RegisterOperation) ([]byte, error) {
	switch op.FuncCode {
	case gridx.FuncCodeReadHoldingRegisters:
		return m.conn.ReadHoldingRegisters(op.Addr, op.Length)

	case gridx.FuncCodeReadInputRegisters:
		return m.conn.ReadInputRegisters(op.Addr, op.Length)

	case gridx.FuncCodeReadCoils:
		return m.conn.ReadCoils(op.Addr, op.Length)

	default:
		return nil, fmt.Errorf("invalid read function code: %d", op.FuncCode)
	}
}

var _ FloatGetter = (*Modbus)(nil)

// FloatGetter implements func() (float64, error)
func (m *Modbus) FloatGetter() (func() (f float64, err error), error) {
	op, err := m.reg.Operation()
	if err != nil {
		return nil, err
	}

	decode, err := m.reg.DecodeFunc()
	if err != nil {
		return nil, err
	}

	return func() (float64, error) {
		bytes, err := m.readBytes(op)
		if err != nil {
			return 0, fmt.Errorf("read failed: %w", err)
		}

		return m.scale * decode(bytes), nil
	}, nil
}

var _ IntGetter = (*Modbus)(nil)

// IntGetter implements IntProvider
func (m *Modbus) IntGetter() (func() (int64, error), error) {
	g, err := m.FloatGetter()

	return func() (int64, error) {
		res, err := g()
		return int64(math.Round(res)), err
	}, err
}

var _ StringGetter = (*Modbus)(nil)

// StringGetter implements StringProvider
func (m *Modbus) StringGetter() (func() (string, error), error) {
	op, err := m.reg.Operation()
	if err != nil {
		return nil, err
	}

	return func() (string, error) {
		b, err := m.readBytes(op)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(bytes.Trim(b, "\x00"))), nil
	}, nil
}

var _ BoolGetter = (*Modbus)(nil)

// BoolGetter implements BoolProvider
func (m *Modbus) BoolGetter() (func() (bool, error), error) {
	g, err := m.FloatGetter()

	return func() (bool, error) {
		res, err := g()
		return res != 0, err
	}, err
}

func (m *Modbus) writeFunc() (func(float64) error, error) {
	op, err := m.reg.Operation()
	if err != nil {
		return nil, err
	}

	encode, err := m.reg.EncodeFunc()
	if err != nil {
		return nil, err
	}

	return func(val float64) error {
		val *= m.scale

		switch op.FuncCode {
		case gridx.FuncCodeWriteSingleCoil:
			var uval uint16
			if val != 0 {
				uval = 0xFF00
			}
			_, err = m.conn.WriteSingleCoil(op.Addr, uval)
			return err

		case gridx.FuncCodeWriteSingleRegister:
			_, err = m.conn.WriteSingleRegister(op.Addr, uint16(val))
			return err

		case gridx.FuncCodeWriteMultipleRegisters:
			b, err := encode(val)
			if err == nil {
				_, err = m.conn.WriteMultipleRegisters(op.Addr, op.Length, b)
			}
			return err

		default:
			return fmt.Errorf("invalid func code: %d", op.FuncCode)
		}
	}, nil
}

var _ FloatSetter = (*Modbus)(nil)

// FloatSetter implements FloatSetter
func (m *Modbus) FloatSetter(_ string) (func(float64) error, error) {
	return m.writeFunc()
}

var _ IntSetter = (*Modbus)(nil)

// IntSetter implements IntSetter
func (m *Modbus) IntSetter(_ string) (func(int64) error, error) {
	fun, err := m.writeFunc()
	if err != nil {
		return nil, err
	}

	return func(val int64) error {
		return fun(float64(val))
	}, nil
}

var _ BoolSetter = (*Modbus)(nil)

// BoolSetter implements BoolSetter
func (m *Modbus) BoolSetter(param string) (func(bool) error, error) {
	set, err := m.IntSetter(param)

	return func(val bool) error {
		var ival int64
		if val {
			ival = 1
		}

		return set(ival)
	}, err
}

var _ BytesSetter = (*Modbus)(nil)

// BytesSetter implements BytesSetter
func (m *Modbus) BytesSetter(_ string) (func([]byte) error, error) {
	op, err := m.reg.Operation()
	if err != nil {
		return nil, err
	}

	return func(val []byte) error {
		switch op.FuncCode {
		case gridx.FuncCodeWriteMultipleRegisters:
			_, err = m.conn.WriteMultipleRegisters(op.Addr, uint16(len(val)/2), val)
			return err

		default:
			return fmt.Errorf("invalid func code: %d", op.FuncCode)
		}
	}, nil
}
