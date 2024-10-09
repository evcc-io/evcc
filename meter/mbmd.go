package meter

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

// ModbusMbmd is an api.Meter implementation with configurable getters and setters.
type ModbusMbmd struct {
	conn     *modbus.Connection
	device   meters.Device
	opPower  modbus.Operation
	opEnergy modbus.Operation
	opSoc    modbus.Operation
}

func init() {
	registry.Add("mbmd", NewModbusMbmdFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateModbusMbmd -b api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhasePowers,Powers,func() (float64, float64, float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64"

// NewModbusMbmdFromConfig creates api.Meter from config
func NewModbusMbmdFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Model              string
		capacity           `mapstructure:",squash"`
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
		if rtu := modbus.IsRS485(cc.Model); rtu {
			cc.RTU = &rtu
		}
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

	log := util.NewLogger("modbus")
	conn.Logger(log.TRACE)

	// prepare device
	device, err := modbus.NewDevice(cc.Model, cc.SubDevice)

	if err == nil {
		err = device.Initialize(conn)

		// silence Kostal implementation errors
		if errors.Is(err, meters.ErrPartiallyOpened) {
			err = nil
		}
	}

	if err != nil {
		return nil, err
	}

	m := &ModbusMbmd{
		conn:   conn,
		device: device,
	}

	m.opPower, err = modbus.ParseOperation(device, cc.Power)
	if err != nil {
		return nil, fmt.Errorf("invalid measurement for power: %s", cc.Power)
	}

	// decorate energy
	var totalEnergy func() (float64, error)
	if cc.Energy != "" {
		m.opEnergy, err = modbus.ParseOperation(device, cc.Energy)
		if err != nil {
			return nil, fmt.Errorf("invalid measurement for energy: %s", cc.Energy)
		}

		totalEnergy = m.totalEnergy
	}

	// decorate currents
	currentsG, err := m.buildPhaseProviders(cc.Currents)
	if err != nil {
		return nil, fmt.Errorf("currents: %w", err)
	}

	// decorate voltages
	voltagesG, err := m.buildPhaseProviders(cc.Voltages)
	if err != nil {
		return nil, fmt.Errorf("voltages: %w", err)
	}

	// decorate powers
	powersG, err := m.buildPhaseProviders(cc.Powers)
	if err != nil {
		return nil, fmt.Errorf("powers: %w", err)
	}

	// decorate soc
	var soc func() (float64, error)
	if cc.Soc != "" {
		m.opSoc, err = modbus.ParseOperation(device, cc.Soc)
		if err != nil {
			return nil, fmt.Errorf("invalid measurement for soc: %s", cc.Soc)
		}

		soc = m.soc
	}

	return decorateModbusMbmd(m, totalEnergy, currentsG, voltagesG, powersG, soc, cc.capacity.Decorator()), nil
}

func (m *ModbusMbmd) buildPhaseProviders(readings []string) (func() (float64, float64, float64, error), error) {
	if len(readings) == 0 {
		return nil, nil
	}

	if len(readings) != 3 {
		return nil, errors.New("need one per phase, total three")
	}

	var phases [3]func() (float64, error)
	for idx, reading := range readings {
		opCurrent, err := modbus.ParseOperation(m.device, reading)
		if err != nil {
			return nil, fmt.Errorf("invalid measurement [%d]: %s", idx, reading)
		}

		phases[idx] = func() (float64, error) {
			return m.floatGetter(opCurrent)
		}
	}

	return collectPhaseProviders(phases), nil
}

// floatGetter executes configured modbus read operation and implements func() (float64, error)
func (m *ModbusMbmd) floatGetter(op modbus.Operation) (float64, error) {
	var res meters.MeasurementResult
	var err error

	if dev, ok := m.device.(*rs485.RS485); ok {
		res, err = dev.QueryOp(m.conn, op.MBMD)
	}

	if dev, ok := m.device.(*sunspec.SunSpec); ok {
		if op.MBMD.IEC61850 != 0 {
			res, err = dev.QueryOp(m.conn, op.MBMD.IEC61850)
		} else {
			res.Value, err = dev.QueryPoint(
				m.conn,
				op.SunSpec.Model,
				op.SunSpec.Block,
				op.SunSpec.Point,
			)
		}
	}

	// silence NaN reading errors by assuming zero
	if err != nil && errors.Is(err, meters.ErrNaN) {
		res.Value = 0
		err = nil
	}

	return res.Value, err
}

// CurrentPower implements the api.Meter interface
func (m *ModbusMbmd) CurrentPower() (float64, error) {
	return m.floatGetter(m.opPower)
}

// totalEnergy implements the api.MeterEnergy interface
func (m *ModbusMbmd) totalEnergy() (float64, error) {
	return m.floatGetter(m.opEnergy)
}

// soc implements the api.Battery interface
func (m *ModbusMbmd) soc() (float64, error) {
	return m.floatGetter(m.opSoc)
}
