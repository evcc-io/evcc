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

// Modbus is an api.Meter implementation with configurable getters and setters.
type Modbus struct {
	conn     *modbus.Connection
	device   meters.Device
	opPower  modbus.Operation
	opEnergy modbus.Operation
	opSoc    modbus.Operation
}

func init() {
	registry.Add("modbus", NewModbusFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateModbus -b api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhasePowers,Powers,func() (float64, float64, float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64"

// NewModbusFromConfig creates api.Meter from config
func NewModbusFromConfig(other map[string]interface{}) (api.Meter, error) {
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
		b := modbus.IsRS485(cc.Model)
		cc.RTU = &b
	}

	conn, err := modbus.NewConnection(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID)
	if err != nil {
		return nil, err
	}

	// set non-default delay
	if cc.Delay > 0 {
		conn.Delay(cc.Delay)
	}

	// set non-default timeout
	if cc.Timeout > 0 {
		conn.Timeout(cc.Timeout)
	}

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

	m := &Modbus{
		conn:   conn,
		device: device,
	}

	if err := modbus.ParseOperation(device, cc.Power, &m.opPower); err != nil {
		return nil, fmt.Errorf("invalid measurement for power: %s", cc.Power)
	}

	// decorate energy
	var totalEnergy func() (float64, error)
	if cc.Energy != "" {
		if err := modbus.ParseOperation(device, cc.Energy, &m.opEnergy); err != nil {
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
		if err := modbus.ParseOperation(device, cc.Soc, &m.opSoc); err != nil {
			return nil, fmt.Errorf("invalid measurement for soc: %s", cc.Soc)
		}

		soc = m.soc
	}

	return decorateModbus(m, totalEnergy, currentsG, voltagesG, powersG, soc, cc.capacity.Decorator()), nil
}

func (m *Modbus) buildPhaseProviders(readings []string) (func() (float64, float64, float64, error), error) {
	var res func() (float64, float64, float64, error)
	if len(readings) > 0 {
		if len(readings) != 3 {
			return nil, errors.New("need one per phase, total three")
		}

		phases := make([]func() (float64, error), 0, 3)
		for idx, reading := range readings {
			var opCurrent modbus.Operation

			if err := modbus.ParseOperation(m.device, reading, &opCurrent); err != nil {
				return nil, fmt.Errorf("invalid measurement [%d]: %s", idx, reading)
			}

			c := func() (float64, error) {
				return m.floatGetter(opCurrent)
			}

			phases = append(phases, c)
		}

		res = collectPhaseProviders(phases)
	}

	return res, nil
}

// floatGetter executes configured modbus read operation and implements func() (float64, error)
func (m *Modbus) floatGetter(op modbus.Operation) (float64, error) {
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
func (m *Modbus) CurrentPower() (float64, error) {
	return m.floatGetter(m.opPower)
}

// totalEnergy implements the api.MeterEnergy interface
func (m *Modbus) totalEnergy() (float64, error) {
	return m.floatGetter(m.opEnergy)
}

// soc implements the api.Battery interface
func (m *Modbus) soc() (float64, error) {
	return m.floatGetter(m.opSoc)
}
