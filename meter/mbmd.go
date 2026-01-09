package meter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/measurement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// Mbmd is an api.Meter implementation with configurable getters and setters.
type Mbmd struct {
	conn   *modbus.Connection
	device *rs485.RS485
}

func init() {
	registry.AddCtx("mbmd", NewMbmdFromConfig)
}

// NewMbmdFromConfig creates api.Meter from config
func NewMbmdFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		Model              string
		batteryCapacity    `mapstructure:",squash"`
		batteryPowerLimits `mapstructure:",squash"`
		batterySocLimits   `mapstructure:",squash"`
		modbus.Settings    `mapstructure:",squash"`
		Power, Energy, Soc string
		Currents           []string
		Voltages           []string
		Powers             []string
		Delay              time.Duration
		Timeout            time.Duration
	}{
		Power: "Power",
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

	powerG, err := mbmd.deviceOp(ops, cc.Power)
	if err != nil {
		return nil, fmt.Errorf("invalid measurement for power: %s", cc.Power)
	}

	// decorate energy
	var totalEnergy func() (float64, error)
	if cc.Energy != "" {
		g, err := mbmd.deviceOp(ops, cc.Energy)
		if err != nil {
			return nil, fmt.Errorf("invalid measurement for energy: %s", cc.Energy)
		}

		totalEnergy = g
	}

	// decorate currents
	currentsG, err := mbmd.buildPhaseProviders(ops, cc.Currents)
	if err != nil {
		return nil, fmt.Errorf("currents: %w", err)
	}

	// decorate voltages
	voltagesG, err := mbmd.buildPhaseProviders(ops, cc.Voltages)
	if err != nil {
		return nil, fmt.Errorf("voltages: %w", err)
	}

	// decorate powers
	powersG, err := mbmd.buildPhaseProviders(ops, cc.Powers)
	if err != nil {
		return nil, fmt.Errorf("powers: %w", err)
	}

	// decorate soc
	var soc func() (float64, error)
	if cc.Soc != "" {
		g, err := mbmd.deviceOp(ops, cc.Soc)
		if err != nil {
			return nil, fmt.Errorf("invalid measurement for soc: %s", cc.Soc)
		}

		soc = g
	}

	m, _ := NewConfigurable(powerG)

	if soc != nil {
		return m.DecorateBattery(totalEnergy, soc, cc.batteryCapacity.Decorator(), cc.batterySocLimits.Decorator(), cc.batteryPowerLimits.Decorator(), nil), nil
	}

	return m.Decorate(totalEnergy, currentsG, voltagesG, powersG, nil), nil
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

func (m *Mbmd) buildPhaseProviders(ops []rs485.Operation, readings []string) (func() (float64, float64, float64, error), error) {
	if len(readings) == 0 {
		return nil, nil
	}

	if len(readings) != 3 {
		return nil, errors.New("need one per phase, total three")
	}

	var phases [3]func() (float64, error)
	for idx, reading := range readings {
		g, err := m.deviceOp(ops, reading)
		if err != nil {
			return nil, fmt.Errorf("invalid measurement [%d]: %s", idx, reading)
		}

		phases[idx] = g
	}

	return measurement.CombinePhases(phases), nil
}
