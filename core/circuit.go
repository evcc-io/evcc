package core

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

// the circuit instances to control the load
type Circuit struct {
	log    *util.Logger
	uiChan chan<- util.Param

	maxCurrent float64   // max allowed current. will not be used when 0
	maxPower   float64   // max allowed power. will not be used when 0
	parent     *Circuit  // parent circuit reference, used to determine current maxs from hierarchy
	meter      api.Meter // meter to determine current power
}

// NewCircuitFromConfig creates a new Circuit
func NewCircuitFromConfig(log *util.Logger, other map[string]interface{}) (*Circuit, error) {
	var cc struct {
		MaxCurrent float64 `mapstructure:"maxCurrent"` // the max allowed current of this circuit
		MaxPower   float64 `mapstructure:"maxPower"`   // the max allowed power of this circuit (kW)
		MeterRef   string  `mapstructure:"meter"`      // Charge meter reference
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	var meter api.Meter
	if cc.MeterRef != "" {
		dev, err := config.Meters().ByName(cc.MeterRef)
		if err != nil {
			return nil, err
		}
		meter = dev.Instance()
	}

	circuit, err := NewCircuit(log, cc.MaxCurrent, cc.MaxPower, meter)
	if err != nil {
		return nil, err
	}

	return circuit, err
}

// NewCircuit creates a circuit
func NewCircuit(log *util.Logger, maxCurrent, maxPower float64, meter api.Meter) (*Circuit, error) {
	circuit := &Circuit{
		log:        log,
		maxCurrent: maxCurrent,
		maxPower:   maxPower,
		meter:      meter,
	}

	if maxCurrent == 0 {
		circuit.log.DEBUG.Printf("validation of max phase current disabled")
	} else if _, ok := meter.(api.PhaseCurrents); !ok {
		return nil, fmt.Errorf("meter does not support phase currents")
	}

	if maxPower == 0 {
		circuit.log.DEBUG.Printf("validation of max power disabled")
	}

	return circuit, nil
}

func (circuit *Circuit) WithParent(parent *Circuit) {
	circuit.parent = parent
}

// // publish sends values to UI and databases
// func (circuit *Circuit) publish(key string, val interface{}) {
// 	// test helper
// 	if circuit.uiChan != nil {
// 		circuit.uiChan <- util.Param{Key: key, Val: val}
// 	}
// }

// // Prepare set the UI channel to publish information
// func (circuit *Circuit) Prepare(uiChan chan<- util.Param) {
// 	circuit.uiChan = uiChan
// 	if circuit.maxCurrent != math.MaxFloat64 {
// 		circuit.publish("maxCurrent", circuit.maxCurrent)
// 	}
// 	if circuit.maxPower != math.MaxFloat64 {
// 		circuit.publish("maxPower", circuit.maxPower)
// 	}
// }

// // update gets called on every site update call.
// // this is used to update the current consumption etc to get published in status and databases
// func (circuit *Circuit) update() error {
// 	if circuit.phaseCurrents != nil {
// 		if _, err := circuit.MaxPhasesCurrent(); err != nil {
// 			return err
// 		}
// 	}
// 	_, err := circuit.CurrentPower()
// 	return err
// }

// var _ Consumer = (*Circuit)(nil)

// // CurrentPower implements consumer interface and determines actual power in use.
// func (circuit *Circuit) CurrentPower() (float64, error) {
// 	return circuit.powerMeter.CurrentPower()
// }

// // GetRemainingPower determines the power left to be used from configured maxPower
// func (circuit *Circuit) GetRemainingPower() float64 {
// 	power, err := circuit.CurrentPower()
// 	if err != nil {
// 		circuit.log.ERROR.Printf("power currents: %v", err)
// 		return 0
// 	}

// 	if circuit.maxPower == math.MaxFloat64 && circuit.parentCircuit == nil {
// 		return circuit.maxPower
// 	}

// 	powerAvailable := circuit.maxPower - power
// 	circuit.publish("overload", powerAvailable < 0)
// 	if powerAvailable < 0 {
// 		circuit.log.WARN.Printf("overload detected - power: %.2fkW, allowed max power is: %.2fkW\n", power/1000, circuit.maxPower/1000)
// 	}

// 	// check parent circuit, return lowest
// 	if circuit.parentCircuit != nil {
// 		powerAvailable = math.Min(powerAvailable, circuit.parentCircuit.GetRemainingPower())
// 	}

// 	if powerAvailable/1000 > 10000.0 {
// 		circuit.log.DEBUG.Printf("circuit using %.2fkW, no max checking", power/1000)
// 	} else {
// 		circuit.log.DEBUG.Printf("circuit using %.2fkW, %.2fkW available", power/1000, powerAvailable/1000)
// 	}

// 	return powerAvailable
// }

// // MaxPhasesCurrent determines current in use. Implements consumer interface
// func (circuit *Circuit) MaxPhasesCurrent() (float64, error) {
// 	if circuit.phaseCurrents == nil {
// 		return 0, fmt.Errorf("no phase meter assigned")
// 	}
// 	i1, i2, i3, err := circuit.phaseCurrents.Currents()
// 	if err != nil {
// 		return 0, fmt.Errorf("failed getting meter currents: %w", err)
// 	}

// 	circuit.log.TRACE.Printf("meter currents: %.3gA", []float64{i1, i2, i3})
// 	circuit.publish("meterCurrents", []float64{i1, i2, i3})

// 	// TODO: phase adjusted handling. Currently we take highest current from all phases
// 	current := math.Max(math.Max(i1, i2), i3)

// 	circuit.log.TRACE.Printf("actual current: %.1fA", current)
// 	circuit.publish("actualCurrent", current)

// 	return current, nil
// }

// // GetRemainingCurrent available current based on max and consumption
// // checks down up to top level parent
// func (circuit *Circuit) GetRemainingCurrent() float64 {
// 	if circuit.maxCurrent == math.MaxFloat64 && circuit.parentCircuit == nil {
// 		return circuit.maxCurrent
// 	}

// 	current, err := circuit.MaxPhasesCurrent()
// 	if err != nil {
// 		circuit.log.ERROR.Printf("max phase currents: %v", err)
// 		return 0
// 	}

// 	curAvailable := circuit.maxCurrent - current
// 	circuit.publish("overload", curAvailable < 0)
// 	if curAvailable < 0 {
// 		circuit.log.WARN.Printf("overload detected - currents: %.1fA, allowed max current is: %.1fA\n", current, circuit.maxCurrent)
// 	}

// 	// check parent circuit, return lowest
// 	if circuit.parentCircuit != nil {
// 		curAvailable = math.Min(curAvailable, circuit.parentCircuit.GetRemainingCurrent())
// 	}
// 	if curAvailable > 10000.0 {
// 		circuit.log.DEBUG.Printf("circuit using %.1fA, no max checking", current)
// 	} else {
// 		circuit.log.DEBUG.Printf("circuit using %.1fA, %.1fA available", current, curAvailable)
// 	}
// 	return curAvailable
// }
