package provider

import (
	"errors"
	"fmt"
	"math"
	"time"

	sunspec "github.com/andig/gosunspec"
	"github.com/andig/gosunspec/typelabel"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters"
	sunsdev "github.com/volkszaehler/mbmd/meters/sunspec"
)

// ModbusSunspec implements modbus RTU and TCP access
type ModbusSunspec struct {
	log    *util.Logger
	conn   *modbus.Connection
	device *sunsdev.SunSpec
	op     modbus.SunSpecOperation
	scale  float64
}

func init() {
	registry.Add("sunspec", NewModbusSunspecFromConfig)
}

// NewModbusSunspecFromConfig creates Modbus plugin
func NewModbusSunspecFromConfig(other map[string]interface{}) (Provider, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		Value           []string
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

	conn, err := modbus.NewConnection(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.Tcp, cc.ID)
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

	log := util.NewLogger("sunspec")
	conn.Logger(log.TRACE)

	if len(cc.Value) == 0 {
		return nil, errors.New("value is required")
	}

	// silence KOSTAL implementation errors
	device := sunsdev.NewDevice("sunspec", cc.SubDevice)
	if err := device.Initialize(conn); err != nil && !errors.Is(err, meters.ErrPartiallyOpened) {
		return nil, err
	}

	var ops []modbus.SunSpecOperation
	for _, val := range cc.Value {
		op, err := modbus.ParsePoint(val)
		if err != nil {
			return nil, fmt.Errorf("invalid sunspec value: %s", cc.Value)
		}
		ops = append(ops, op)
	}

	mb := &ModbusSunspec{
		log:    log,
		conn:   conn,
		device: device,
		scale:  cc.Scale,
	}

	for _, op := range ops {
		if _, _, err := device.QueryPointAny(conn, op.Model, op.Block, op.Point); err == nil {
			mb.op = op
			return mb, nil
		}
	}

	return nil, fmt.Errorf("sunspec model not found: %v", ops)
}

func (m *ModbusSunspec) floatGetter() (f float64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	res, err := m.device.QueryPoint(
		m.conn,
		m.op.Model,
		m.op.Block,
		m.op.Point,
	)
	if err != nil && !errors.Is(err, meters.ErrNaN) {
		return 0, fmt.Errorf("model %d block %d point %s: %w", m.op.Model, m.op.Block, m.op.Point, err)
	}

	m.log.TRACE.Printf("model %d block %d point %s: %v", m.op.Model, m.op.Block, m.op.Point, res)

	return m.scale * res, nil
}

var _ FloatProvider = (*Modbus)(nil)

// FloatGetter executes configured modbus read operation and implements func() (float64, error)
func (m *ModbusSunspec) FloatGetter() (func() (f float64, err error), error) {
	return m.floatGetter, nil
}

var _ IntProvider = (*Modbus)(nil)

// IntGetter executes configured modbus read operation and implements IntProvider
func (m *ModbusSunspec) IntGetter() (func() (int64, error), error) {
	g, err := m.FloatGetter()

	return func() (int64, error) {
		res, err := g()
		return int64(math.Round(res)), err
	}, err
}

func (m *ModbusSunspec) blockPoint() (block sunspec.Block, point sunspec.Point, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	block, point, err = m.device.QueryPointAny(
		m.conn,
		m.op.Model,
		m.op.Block,
		m.op.Point,
	)
	if err != nil {
		err = fmt.Errorf("model %d block %d point %s: %w", m.op.Model, m.op.Block, m.op.Point, err)
	}

	return block, point, err
}

// TODO scale factors

var _ SetFloatProvider = (*Modbus)(nil)

// FloatSetter executes configured modbus write operation and implements SetFloatProvider
func (m *ModbusSunspec) FloatSetter(_ string) (func(float64) error, error) {
	block, point, err := m.blockPoint()
	if err != nil {
		return nil, err
	}

	typ := point.Type()

	return func(val float64) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()

		val = val * m.scale
		switch typ {
		case typelabel.Float32:
			point.SetFloat32(float32(val))
		default:
			return fmt.Errorf("invalid point type: %s", typ)
		}

		return block.Write(m.op.Point)
	}, nil
}

var _ SetIntProvider = (*Modbus)(nil)

// IntSetter executes configured modbus write operation and implements SetIntProvider
func (m *ModbusSunspec) IntSetter(_ string) (func(int64) error, error) {
	block, point, err := m.blockPoint()
	if err != nil {
		return nil, err
	}

	typ := point.Type()

	return func(val int64) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()

		val = int64(float64(val) * m.scale)

		switch typ {
		case typelabel.Bitfield16:
			point.SetBitfield16(sunspec.Bitfield16(val))
		case typelabel.Bitfield32:
			point.SetBitfield32(sunspec.Bitfield32(val))
		case typelabel.Enum16:
			point.SetEnum16(sunspec.Enum16(val))
		case typelabel.Enum32:
			point.SetEnum32(sunspec.Enum32(val))
		case typelabel.Int16:
			point.SetInt16(int16(val))
		case typelabel.Int32:
			point.SetInt32(int32(val))
		case typelabel.Int64:
			point.SetInt64(val)
		case typelabel.Uint16:
			point.SetUint16(uint16(val))
		case typelabel.Uint32:
			point.SetUint32(uint32(val))
		case typelabel.Uint64:
			point.SetUint64(uint64(val))
		default:
			return fmt.Errorf("invalid point type: %s", typ)
		}

		return block.Write(m.op.Point)
	}, nil
}
