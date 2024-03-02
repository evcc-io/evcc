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
	reg   modbus.Register
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

	mb := &Modbus{
		log:   log,
		conn:  conn,
		reg:   cc.Register,
		scale: cc.Scale,
	}
	return mb, nil
}

func (m *Modbus) bytesGetter(op modbus.RegisterOperation) ([]byte, error) {
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

func (m *Modbus) floatGetter(op modbus.RegisterOperation) (f float64, err error) {
	bytes, err := m.bytesGetter(op)
	if err != nil {
		return 0, fmt.Errorf("read failed: %w", err)
	}

	return m.scale * op.Decode(bytes), nil
}

var _ FloatProvider = (*Modbus)(nil)

// FloatGetter implements func() (float64, error)
func (m *Modbus) FloatGetter() (func() (f float64, err error), error) {
	op, err := m.reg.Operation()
	if err != nil {
		return nil, err
	}

	return func() (float64, error) {
		return m.floatGetter(op)
	}, nil
}

var _ IntProvider = (*Modbus)(nil)

// IntGetter implements IntProvider
func (m *Modbus) IntGetter() (func() (int64, error), error) {
	g, err := m.FloatGetter()

	return func() (int64, error) {
		res, err := g()
		return int64(math.Round(res)), err
	}, err
}

var _ StringProvider = (*Modbus)(nil)

// StringGetter implements StringProvider
func (m *Modbus) StringGetter() (func() (string, error), error) {
	op, err := m.reg.Operation()
	if err != nil {
		return nil, err
	}

	return func() (string, error) {
		b, err := m.bytesGetter(op)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(bytes.TrimLeft(b, "\x00"))), nil
	}, nil
}

var _ BoolProvider = (*Modbus)(nil)

// BoolGetter implements BoolProvider
func (m *Modbus) BoolGetter() (func() (bool, error), error) {
	op, err := m.reg.Operation()
	if err != nil {
		return nil, err
	}

	return func() (bool, error) {
		bytes, err := m.bytesGetter(op)
		if err != nil {
			return false, err
		}

		u, err := UintFromBytes(bytes)
		return u > 0, err
	}, nil
}

func (m *Modbus) writeMultipleRegisters(op modbus.RegisterOperation, val uint64) error {
	val = op.Encode(val)
	fmt.Printf("encode:\t% x\n", val)

	var err error
	switch op.Length {
	case 1:
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], uint16(val))
		_, err = m.conn.WriteMultipleRegisters(op.Addr, 1, b[:])

	case 2:
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(val))
		_, err = m.conn.WriteMultipleRegisters(op.Addr, 2, b[:])

	case 4:
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], val)
		_, err = m.conn.WriteMultipleRegisters(op.Addr, 4, b[:])

	default:
		err = fmt.Errorf("invalid write length: %d", op.Length)
	}

	return err
}

var _ SetFloatProvider = (*Modbus)(nil)

// FloatSetter implements SetFloatProvider
func (m *Modbus) FloatSetter(_ string) (func(float64) error, error) {
	op, err := m.reg.Operation()
	if err != nil {
		return nil, err
	}

	// need multiple registers for float
	if op.FuncCode != gridx.FuncCodeWriteMultipleRegisters {
		return nil, fmt.Errorf("invalid write function code: %d", op.FuncCode)
	}

	return func(val float64) error {
		val = m.scale * val
		fmt.Printf("val:\t%v\n", val)

		var uval uint64
		switch op.Length {
		case 2:
			fmt.Printf("Float32bits:\t% x\n", math.Float32bits(float32(val)))
			uval = uint64(math.Float32bits(float32(val)))
			fmt.Printf("uval:\t% x\n", uval)
		case 4:
			uval = math.Float64bits(val)
		}

		var err error
		switch op.FuncCode {
		case gridx.FuncCodeWriteMultipleRegisters:
			err = m.writeMultipleRegisters(op, uval)

		default:
			err = fmt.Errorf("invalid write function code: %d", op.FuncCode)
		}

		return err
	}, nil
}

var _ SetIntProvider = (*Modbus)(nil)

// IntSetter implements SetIntProvider
func (m *Modbus) IntSetter(_ string) (func(int64) error, error) {
	op, err := m.reg.Operation()
	if err != nil {
		return nil, err
	}

	return func(val int64) error {
		ival := int64(m.scale * float64(val))

		var err error
		switch op.FuncCode {
		case gridx.FuncCodeWriteSingleCoil:
			if ival != 0 {
				// Modbus protocol requires 0xFF00 for ON
				// and 0x0000 for OFF
				ival = 0xFF00
			}
			_, err = m.conn.WriteSingleCoil(op.Addr, uint16(ival))

		case gridx.FuncCodeWriteSingleRegister:
			_, err = m.conn.WriteSingleRegister(op.Addr, uint16(ival))

		case gridx.FuncCodeWriteMultipleRegisters:
			uval := uint64(ival)
			switch {
			case op.Length == 2:
			}
			err = m.writeMultipleRegisters(op, uval)

		default:
			err = fmt.Errorf("invalid write function code: %d", op.FuncCode)
		}

		return err
	}, nil
}

var _ SetBoolProvider = (*Modbus)(nil)

// BoolSetter implements SetBoolProvider
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
