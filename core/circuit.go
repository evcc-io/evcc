package core

import (
	"fmt"
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// the circuit instances to control the load
type Circuit struct {
	log    *util.Logger
	uiChan chan<- util.Param

	maxCurrent    float64           // max allowed current. will not be used when 0
	maxPower      float64           // max allowed power. will not be used when 0
	parentCircuit *Circuit          // parent circuit reference, used to determine current limits from hierarchy
	phaseCurrents api.PhaseCurrents // meter to determine phase current
	powerMeter    api.Meter         // meter to determine current power
}

// NewCircuitFromConfig creates a new Circuit
func NewCircuitFromConfig(log *util.Logger, cp configProvider, other map[string]interface{}) (*Circuit, *VMeter, string, error) {
	var cc struct {
		Name       string  `mapstructure:"name"`       // unique name, used as reference in lp
		MaxCurrent float64 `mapstructure:"maxCurrent"` // the max allowed current of this circuit
		MaxPower   float64 `mapstructure:"maxPower"`   // the max allowed power of this circuit (kW)
		MeterRef   string  `mapstructure:"meter"`      // Charge meter reference
		ParentRef  string  `mapstructure:"parent"`     // name of parent circuit
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, nil, "", err
	}

	var err error

	if cc.Name == "" {
		return nil, nil, "", fmt.Errorf("failed configuring circuit, need to have a name")
	}

	if c, _ := cp.Circuit(cc.Name); c != nil {
		return nil, nil, "", fmt.Errorf("failed to create circuit, name already used: %s", cc.Name)
	}

	var parent *Circuit
	if cc.ParentRef != "" {
		if parent, err = cp.Circuit(cc.ParentRef); err != nil {
			return nil, nil, "", err
		}
	}

	var vMeter *VMeter
	var meterPhaseCurrents api.PhaseCurrents
	var meterPower api.Meter

	if cc.MeterRef != "" {
		var m api.Meter
		if m, err = cp.Meter(cc.MeterRef); err != nil {
			return nil, nil, "", err
		}

		meterPower = m

		var ok bool
		if meterPhaseCurrents, ok = m.(api.PhaseCurrents); !ok {
			if cc.MaxCurrent != 0 {
				return nil, nil, "", fmt.Errorf("meter %s does not support phase currents, but current checking configured", cc.MeterRef)
			}
		}
	} else {
		// no dedicated vMeter given, create vmeter
		vMeter = NewVMeter(cc.Name)
		meterPhaseCurrents = vMeter
		meterPower = vMeter
	}

	newCC := NewCircuit(log, cc.MaxCurrent, cc.MaxPower*1000, parent, meterPhaseCurrents, meterPower)

	if cc.ParentRef != "" {
		if vm := cp.VMeter(cc.ParentRef); vm != nil {
			vm.AddConsumer(newCC)
		}
	}
	return newCC, vMeter, cc.Name, nil
}

// NewCircuit a circuit with defaults
func NewCircuit(log *util.Logger, limitCurrent float64, limitPower float64, p *Circuit, pc api.PhaseCurrents, pm api.Meter) *Circuit {
	if pm == nil {
		// we always need a power meter
		log.ERROR.Printf("power meter not allowed to be nil")
		return nil
	}
	circuit := &Circuit{
		log:           log,
		maxCurrent:    limitCurrent,
		maxPower:      limitPower,
		parentCircuit: p,
		phaseCurrents: pc,
		powerMeter:    pm,
	}

	// if either MaxCurrent or MaxPower is 0, consider to be not relevant. Set to MaxFloat to never hit a limit
	// support both to be 0 for logging purposes (e.g. use it to log current aggregatated consumption to influx)
	if limitCurrent == 0 {
		circuit.maxCurrent = math.MaxFloat64
		circuit.log.DEBUG.Printf("current checking disabled")
	} else {
		if pc == nil {
			log.ERROR.Printf("need phase meter when current checking is enabled")
			return nil
		}
	}
	if limitPower == 0 {
		circuit.maxPower = math.MaxFloat64
		circuit.log.DEBUG.Printf("power checking disabled")
	}
	return circuit
}

// publish sends values to UI and databases
func (circuit *Circuit) publish(key string, val interface{}) {
	// test helper
	if circuit.uiChan != nil {
		circuit.uiChan <- util.Param{Key: key, Val: val}
	}
}

// Prepare set the UI channel to publish information
func (circuit *Circuit) Prepare(uiChan chan<- util.Param) {
	circuit.uiChan = uiChan
	if circuit.maxCurrent != math.MaxFloat64 {
		circuit.publish("maxCurrent", circuit.maxCurrent)
	}
	if circuit.maxPower != math.MaxFloat64 {
		circuit.publish("maxPower", circuit.maxPower)
	}
}

// update gets called on every site update call.
// this is used to update the current consumption etc to get published in status and databases
func (circuit *Circuit) update() error {
	if circuit.phaseCurrents != nil {
		if _, err := circuit.MaxPhasesCurrent(); err != nil {
			return err
		}
	}
	_, err := circuit.CurrentPower()
	return err
}

var _ Consumer = (*Circuit)(nil)

// MaxPower determines current in use. Implements consumer interface
func (circuit *Circuit) CurrentPower() (float64, error) {
	return circuit.powerMeter.CurrentPower()
}

// GetRemainingPower determines the power left to be used from confgigured maxPower
func (circuit *Circuit) GetRemainingPower() float64 {

	power, err := circuit.CurrentPower()
	if err != nil {
		circuit.log.ERROR.Printf("power currents: %v", err)
		return 0
	}

	if circuit.maxPower == math.MaxFloat64 && circuit.parentCircuit == nil {
		return circuit.maxPower
	}

	powerAvailable := circuit.maxPower - power
	circuit.publish("overload", powerAvailable < 0)
	if powerAvailable < 0 {
		circuit.log.WARN.Printf("overload detected - power: %.2fkW, allowed max power is: %.2fkW\n", power/1000, circuit.maxPower/1000)
	}

	// check parent circuit, return lowest
	if circuit.parentCircuit != nil {
		powerAvailable = math.Min(powerAvailable, circuit.parentCircuit.GetRemainingPower())
	}

	if powerAvailable/1000 > 10000.0 {
		circuit.log.DEBUG.Printf("circuit using %.2fkW, no limit checking", power/1000)
	} else {
		circuit.log.DEBUG.Printf("circuit using %.2fkW, %.2fkW available", power/1000, powerAvailable/1000)

	}

	return powerAvailable
}

// MaxPhasesCurrent determines current in use. Implements consumer interface
func (circuit *Circuit) MaxPhasesCurrent() (float64, error) {
	if circuit.phaseCurrents == nil {
		return 0, fmt.Errorf("no phase meter assigned")
	}
	i1, i2, i3, err := circuit.phaseCurrents.Currents()
	if err != nil {
		return 0, fmt.Errorf("failed getting meter currents: %w", err)
	}

	circuit.log.TRACE.Printf("meter currents: %.3gA", []float64{i1, i2, i3})
	circuit.publish("meterCurrents", []float64{i1, i2, i3})

	// TODO: phase adjusted handling. Currently we take highest current from all phases
	current := math.Max(math.Max(i1, i2), i3)

	circuit.log.TRACE.Printf("actual current: %.1fA", current)
	circuit.publish("actualCurrent", current)

	return current, nil
}

// GetRemainingCurrent available current based on limit and consumption
// checks down up to top level parent
func (circuit *Circuit) GetRemainingCurrent() float64 {

	if circuit.maxCurrent == math.MaxFloat64 && circuit.parentCircuit == nil {
		return circuit.maxCurrent
	}

	current, err := circuit.MaxPhasesCurrent()
	if err != nil {
		circuit.log.ERROR.Printf("max phase currents: %v", err)
		return 0
	}

	curAvailable := circuit.maxCurrent - current
	circuit.publish("overload", curAvailable < 0)
	if curAvailable < 0 {
		circuit.log.WARN.Printf("overload detected - currents: %.1fA, allowed max current is: %.1fA\n", current, circuit.maxCurrent)
	}

	// check parent circuit, return lowest
	if circuit.parentCircuit != nil {
		curAvailable = math.Min(curAvailable, circuit.parentCircuit.GetRemainingCurrent())
	}
	if curAvailable > 10000.0 {
		circuit.log.DEBUG.Printf("circuit using %.1fA, no limit checking", current)
	} else {
		circuit.log.DEBUG.Printf("circuit using %.1fA, %.1fA available", current, curAvailable)
	}
	return curAvailable
}
