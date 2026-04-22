package plugin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// Mbmd is an api.Meter implementation with configurable getters and setters.
type Mbmd struct {
	conn   *modbus.Connection
	device *rs485.RS485
	getter func() (float64, error)
}

func init() {
	registry.AddCtx("mbmd", NewMbmdFromConfig)
}

// NewMbmdFromConfig creates api.Meter from config
func NewMbmdFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		modbus.Settings
		Model   string
		Value   string
		Delay   time.Duration
		Timeout time.Duration
	}{
		Settings: modbus.Settings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// assume RTU if not set and this is a known RS485 meter model
	if cc.RTU == nil {
		if rtu := isRS485(cc.Model); rtu {
			cc.RTU = &rtu
		}
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

	log := util.NewLogger("modbus")
	conn.Logger(log.TRACE)

	// prepare device
	device, err := rs485.NewDevice(strings.ToUpper(cc.Model))
	if err != nil {
		return nil, err
	}

	mbmd := &Mbmd{
		conn:   conn,
		device: device,
	}

	ops := device.Producer().Produce()

	getter, err := mbmd.deviceOp(ops, cc.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid measurement: %s", cc.Value)
	}

	mbmd.getter = getter

	return mbmd, nil
}

// deviceOp checks is RS485 device supports operation
func (m *Mbmd) deviceOp(ops []rs485.Operation, name string) (func() (float64, error), error) {
	// leading minus sign?
	name, invert := strings.CutPrefix(name, "-")

	measurement, err := meters.MeasurementString(name)
	if err != nil {
		return nil, fmt.Errorf("invalid measurement: %s", name)
	}

	for _, op := range ops {
		if op.IEC61850 == measurement {
			return func() (float64, error) {
				res, err := m.device.QueryOp(m.conn, op)

				// silence NaN reading errors by assuming zero
				if err != nil && errors.Is(err, meters.ErrNaN) {
					res.Value = 0
					err = nil
				}

				if invert {
					return -res.Value, err
				}

				return res.Value, err
			}, nil
		}
	}

	return nil, fmt.Errorf("unsupported measurement: %s", measurement.String())
}
