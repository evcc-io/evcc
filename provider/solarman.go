package provider

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/solarman"
	gridx "github.com/grid-x/modbus"
)

type Solarman struct {
	conn  *solarman.Connection
	reg   modbus.Register
	scale float64
}

func init() {
	registry.Add("solarman", NewSolarmanFromConfig)
}

func NewSolarmanFromConfig(other map[string]interface{}) (Provider, error) {
	cc := struct {
		solarman.Settings `mapstructure:",squash"`
		Register          modbus.Register
		Scale             float64
	}{
		Scale: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	solarman.Lock()
	defer solarman.Unlock()
	uri := net.JoinHostPort(cc.Host, strconv.Itoa(cc.Port))

	conn, err := solarman.NewConnection(uri, cc.Loggerserial, cc.Slaveid)
	if err != nil {
		return nil, err
	}

	sm := &Solarman{
		conn:  conn,
		reg:   cc.Register,
		scale: cc.Scale,
	}

	return sm, nil
}

func (sm *Solarman) readBytes(op modbus.RegisterOperation) ([]byte, error) {
	switch op.FuncCode {
	case gridx.FuncCodeReadHoldingRegisters:
		return sm.conn.ReadHoldingRegisters(op.Addr, op.Length)

	case gridx.FuncCodeReadInputRegisters:
		return sm.conn.ReadInputRegisters(op.Addr, op.Length)

	case gridx.FuncCodeReadCoils:
		return sm.conn.ReadCoils(op.Addr, op.Length)

	default:
		return nil, fmt.Errorf("invalid read function code: %d", op.FuncCode)
	}
}

var _ FloatProvider = (*Solarman)(nil)

func (sm *Solarman) FloatGetter() (func() (f float64, err error), error) {
	op, err := sm.reg.Operation()
	if err != nil {
		return nil, err
	}
	decode, err := sm.reg.DecodeFunc()
	if err != nil {
		return nil, err
	}

	return func() (float64, error) {
		bytes, err := sm.readBytes(op)
		if err != nil {
			return 0, fmt.Errorf("read failed: %w", err)
		}
		return sm.scale * decode(bytes), nil
	}, nil
}

var _ IntProvider = (*Solarman)(nil)

func (sm *Solarman) IntGetter() (func() (int64, error), error) {
	g, err := sm.FloatGetter()

	return func() (int64, error) {
		res, err := g()
		return int64(math.Round(res)), err
	}, err
}

var _ StringProvider = (*Solarman)(nil)

func (sm *Solarman) StringGetter() (func() (string, error), error) {
	op, err := sm.reg.Operation()
	if err != nil {
		return nil, err
	}

	return func() (string, error) {
		b, err := sm.readBytes(op)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(bytes.TrimLeft(b, "\x00"))), nil
	}, nil
}

var _ BoolProvider = (*Modbus)(nil)

func (sm *Solarman) BoolGetter() (func() (bool, error), error) {
	g, err := sm.FloatGetter()

	return func() (bool, error) {
		res, err := g()
		return res != 0, err
	}, err
}

func (sm *Solarman) writeFunc() (func(float64) error, error) {
	op, err := sm.reg.Operation()
	if err != nil {
		return nil, err
	}

	encode, err := sm.reg.EncodeFunc()
	if err != nil {
		return nil, err
	}

	return func(val float64) error {
		val *= sm.scale

		switch op.FuncCode {
		case gridx.FuncCodeWriteSingleCoil:
			var uval uint16
			if val != 0 {
				uval = 0xFF00
			}
			_, err = sm.conn.WriteSingleCoil(op.Addr, uval)
			return err

		case gridx.FuncCodeWriteSingleRegister:
			_, err = sm.conn.WriteSingleRegister(op.Addr, uint16(val))
			return err

		case gridx.FuncCodeWriteMultipleRegisters:
			b, err := encode(val)
			if err == nil {
				_, err = sm.conn.WriteMultipleRegisters(op.Addr, op.Length, b)
			}
			return err

		default:
			return fmt.Errorf("invalid func code: %d", op.FuncCode)
		}
	}, nil
}

var _ SetFloatProvider = (*Solarman)(nil)

func (m *Solarman) FloatSetter(_ string) (func(float64) error, error) {
	return m.writeFunc()
}

var _ SetIntProvider = (*Modbus)(nil)

func (m *Solarman) IntSetter(_ string) (func(int64) error, error) {
	fun, err := m.writeFunc()
	if err != nil {
		return nil, err
	}

	return func(val int64) error {
		return fun(float64(val))
	}, nil
}

var _ SetBoolProvider = (*Modbus)(nil)

func (m *Solarman) BoolSetter(param string) (func(bool) error, error) {
	set, err := m.IntSetter(param)

	return func(val bool) error {
		var ival int64
		if val {
			ival = 1
		}

		return set(ival)
	}, err
}
