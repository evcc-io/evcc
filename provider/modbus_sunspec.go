package provider

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

// ModbusSunspec implements modbus RTU and TCP access
type ModbusSunspec struct {
	log    *util.Logger
	conn   *modbus.Connection
	device *sunspec.SunSpec
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

	if cc.Value == "" {
		return nil, errors.New("value is required")
	}

	// silence KOSTAL implementation errors
	device := sunspec.NewDevice("sunspec", cc.SubDevice)
	if err := device.Initialize(conn); err != nil && !errors.Is(err, meters.ErrPartiallyOpened) {
		return nil, err
	}

	var op modbus.SunSpecOperation
	op.Model, op.Block, op.Point, err = modbus.ParsePoint(cc.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid sunspec value: %s", cc.Value)
	}

	mb := &ModbusSunspec{
		log:    log,
		conn:   conn,
		device: device,
		op:     op,
		scale:  cc.Scale,
	}
	return mb, nil
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
	if err != nil {
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

// var _ SetFloatProvider = (*Modbus)(nil)

// // FloatSetter executes configured modbus write operation and implements SetFloatProvider
// func (m *ModbusSunspec) FloatSetter(_ string) (func(float64) error, error) {
// }

// var _ SetIntProvider = (*Modbus)(nil)

// // IntSetter executes configured modbus write operation and implements SetIntProvider
// func (m *ModbusSunspec) IntSetter(_ string) (func(int64) error, error) {
// }
