package plugin

import (
	"context"
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
	registry.AddCtx("sunspec", NewModbusSunspecFromConfig)
}

// NewModbusSunspecFromConfig creates Modbus plugin
func NewModbusSunspecFromConfig(ctx context.Context, other map[string]interface{}) (Plugin, error) {
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

	modbus.Lock()
	defer modbus.Unlock()

	conn, err := modbus.NewConnection(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Settings.Protocol(), cc.ID)
	if err != nil {
		return nil, err
	}

	// set non-default timeout
	conn.Timeout(cc.Timeout)

	// set non-default delay
	conn.Delay(cc.Delay)

	// set non-default connect delay
	conn.ConnectDelay(cc.ConnectDelay)

	log := util.NewLogger("sunspec")
	conn.Logger(log.TRACE)

	if len(cc.Value) == 0 {
		return nil, errors.New("value is required")
	}

	devices := sunspecDevices.Get(conn)
	if devices == nil {
		devices, err = sunsdev.DeviceTree(conn)
		if err != nil && !errors.Is(err, meters.ErrPartiallyOpened) {
			return nil, err
		}

		sunspecDevices.Put(conn, devices)
	}

	device := sunspecSubDevices.Get(conn, cc.SubDevice)
	if device == nil {
		// silence KOSTAL implementation errors
		device = sunsdev.NewDevice("sunspec", cc.SubDevice)
		if err := device.InitializeWithTree(devices); err != nil {
			return nil, err
		}

		sunspecSubDevices.Put(conn, cc.SubDevice, device)
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

var _ FloatGetter = (*Modbus)(nil)

// FloatGetter executes configured modbus read operation and implements func() (float64, error)
func (m *ModbusSunspec) FloatGetter() (func() (f float64, err error), error) {
	return m.floatGetter, nil
}

var _ IntGetter = (*Modbus)(nil)

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

var _ FloatSetter = (*Modbus)(nil)

// FloatSetter executes configured modbus write operation and implements FloatSetter
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

var _ IntSetter = (*Modbus)(nil)

// IntSetter executes configured modbus write operation and implements IntSetter
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

		// SetValue is used to include the scale factor when writing
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
			point.SetValue(int16(val))
		case typelabel.Int32:
			point.SetValue(int32(val))
		case typelabel.Int64:
			point.SetValue(val)
		case typelabel.Uint16:
			point.SetValue(uint16(val))
		case typelabel.Uint32:
			point.SetValue(uint32(val))
		case typelabel.Uint64:
			point.SetValue(uint64(val))
		default:
			return fmt.Errorf("invalid point type: %s", typ)
		}

		return block.Write(m.op.Point)
	}, nil
}
