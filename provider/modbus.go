package provider

import (
	"bytes"
	"encoding/binary"
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
	op    modbus.RegisterOperation
	scale float64
}

func init() {
	registry.Add("modbus", NewModbusFromConfig)
}

// NewModbusFromConfig creates Modbus plugin
func NewModbusFromConfig(other map[string]interface{}) (Provider, error) {
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

	if err := cc.Register.Error(); err != nil {
		return nil, err
	}

	op, err := cc.Register.Operation()
	if err != nil {
		return nil, err
	}

	mb := &Modbus{
		log:   log,
		conn:  conn,
		op:    op,
		scale: cc.Scale,
	}
	return mb, nil
}

func (m *Modbus) bytesGetter() ([]byte, error) {
	switch m.op.FuncCode {
	case gridx.FuncCodeReadHoldingRegisters:
		return m.conn.ReadHoldingRegisters(m.op.Addr, m.op.Length)

	case gridx.FuncCodeReadInputRegisters:
		return m.conn.ReadInputRegisters(m.op.Addr, m.op.Length)

	case gridx.FuncCodeReadCoils:
		return m.conn.ReadCoils(m.op.Addr, m.op.Length)

	default:
		return nil, fmt.Errorf("invalid read function code: %d", m.op.FuncCode)
	}
}

func (m *Modbus) floatGetter() (f float64, err error) {
	bytes, err := m.bytesGetter()
	if err != nil {
		return 0, fmt.Errorf("read failed: %w", err)
	}

	return m.scale * m.op.Decode(bytes), nil
}

var _ FloatProvider = (*Modbus)(nil)

// FloatGetter executes configured modbus read operation and implements func() (float64, error)
func (m *Modbus) FloatGetter() (func() (f float64, err error), error) {
	return m.floatGetter, nil
}

var _ IntProvider = (*Modbus)(nil)

// IntGetter executes configured modbus read operation and implements IntProvider
func (m *Modbus) IntGetter() (func() (int64, error), error) {
	g, err := m.FloatGetter()

	return func() (int64, error) {
		res, err := g()
		return int64(math.Round(res)), err
	}, err
}

var _ StringProvider = (*Modbus)(nil)

// StringGetter executes configured modbus read operation and implements IntProvider
func (m *Modbus) StringGetter() (func() (string, error), error) {
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
func (m *Modbus) BoolGetter() (func() (bool, error), error) {
	return func() (bool, error) {
		bytes, err := m.bytesGetter()
		if err != nil {
			return false, err
		}

		u, err := UintFromBytes(bytes)
		return u > 0, err
	}, nil
}

var _ SetFloatProvider = (*Modbus)(nil)

func (m *Modbus) writeMultipleRegisters(val uint64) error {
	val = m.op.Encode(val)

	var err error
	switch m.op.Length {
	case 1:
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], uint16(val))
		_, err = m.conn.WriteMultipleRegisters(m.op.Addr, 1, b[:])

	case 2:
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(val))
		_, err = m.conn.WriteMultipleRegisters(m.op.Addr, 2, b[:])

	case 4:
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], val)
		_, err = m.conn.WriteMultipleRegisters(m.op.Addr, 4, b[:])

	default:
		err = fmt.Errorf("invalid write length: %d", m.op.Length)
	}

	return err
}

// FloatSetter executes configured modbus write operation and implements SetFloatProvider
func (m *Modbus) FloatSetter(_ string) (func(float64) error, error) {
	// need multiple registers for float
	if m.op.FuncCode != gridx.FuncCodeWriteMultipleRegisters {
		return nil, fmt.Errorf("invalid write function code: %d", m.op.FuncCode)
	}

	return func(val float64) error {
		val = m.scale * val

		var uval uint64
		switch m.op.Length {
		case 2:
			uval = uint64(math.Float32bits(float32(val)))
		case 4:
			uval = math.Float64bits(val)
		}

		var err error
		switch m.op.FuncCode {
		case gridx.FuncCodeWriteMultipleRegisters:
			err = m.writeMultipleRegisters(uval)

		default:
			err = fmt.Errorf("invalid write function code: %d", m.op.FuncCode)
		}

		return err
	}, nil
}

var _ SetIntProvider = (*Modbus)(nil)

// IntSetter executes configured modbus write operation and implements SetIntProvider
func (m *Modbus) IntSetter(_ string) (func(int64) error, error) {
	return func(val int64) error {
		ival := int64(m.scale * float64(val))

		var err error
		switch m.op.FuncCode {
		case gridx.FuncCodeWriteSingleRegister:
			_, err = m.conn.WriteSingleRegister(m.op.Addr, uint16(ival))

		case gridx.FuncCodeWriteMultipleRegisters:
			err = m.writeMultipleRegisters(uint64(ival))

		case gridx.FuncCodeWriteSingleCoil:
			if ival != 0 {
				// Modbus protocol requires 0xFF00 for ON
				// and 0x0000 for OFF
				ival = 0xFF00
			}
			_, err = m.conn.WriteSingleCoil(m.op.Addr, uint16(ival))

		default:
			err = fmt.Errorf("invalid write function code: %d", m.op.FuncCode)
		}

		return err
	}, nil
}

var _ SetBoolProvider = (*Modbus)(nil)

// BoolSetter executes configured modbus write operation and implements SetBoolProvider
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
