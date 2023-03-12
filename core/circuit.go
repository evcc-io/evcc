package core

import (
	"fmt"
	"math"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// struct to setup the circuit hierarchy
type CircuitConfig struct {
	Name       string  `mapstructure:"name"`       // unique name, used as reference in lp
	MaxCurrent float64 `mapstructure:"maxCurrent"` // the max allowed current of this circuit
	MeterRef   string  `mapstructure:"meter"`      // Charge meter reference
	ParentRef  string  `mapstructure:"parent"`     // name of parent circuit
}

// the circuit instances to control the load
type Circuit struct {
	log    *util.Logger
	uiChan chan<- util.Param

	maxCurrent    float64           // max allowed current
	ParentCircuit *Circuit          // parent circuit reference, used to determine current limits from hierarchy
	PhaseCurrents api.PhaseCurrents // meter to determine phase current
}

// GetCurrent determines current in use. Implements consumer interface
func (circuit *Circuit) MaxPhasesCurrent() (float64, error) {
	var current float64

	i1, i2, i3, err := circuit.PhaseCurrents.Currents()
	if err != nil {
		return 0, fmt.Errorf("failed getting meter currents: %w", err)
	}
	circuit.log.TRACE.Printf("meter currents: %.3gA", []float64{i1, i2, i3})
	circuit.publish("meterCurrents", []float64{i1, i2, i3})
	// TODO: phase adjusted handling. Currently we take highest current from all phases
	current = math.Max(math.Max(i1, i2), i3)

	circuit.log.TRACE.Printf("actual current: %.1fA", current)
	circuit.publish("actualCurrent", current)
	return current, nil
}

// GetRemainingCurrent avaialble current based on limit and consumption
// checks down up to top level parent
func (circuit *Circuit) GetRemainingCurrent() float64 {
	// first update current current, mainly to regularly publish the value
	current, err := circuit.MaxPhasesCurrent()
	if err != nil {
		circuit.log.ERROR.Printf("max phase currents: %v", err)
		return 0
	}
	curAvailable := circuit.maxCurrent - current
	if curAvailable < 0.0 {
		circuit.log.WARN.Printf("overload detected - currents: %.1fA, allowed max current is: %.1fA\n", current, circuit.maxCurrent)
		circuit.publish("overload", true)
	} else {
		circuit.publish("overload", false)
	}
	// check parent circuit, return lowest
	if circuit.ParentCircuit != nil {
		curAvailable = math.Min(curAvailable, circuit.ParentCircuit.GetRemainingCurrent())
	}
	circuit.log.DEBUG.Printf("circuit using %.1fA, %.1fA available", current, curAvailable)
	return curAvailable
}

// NewCircuit a circuit with defaults
func NewCircuit(limit float64, p *Circuit, l *util.Logger) *Circuit {
	circuit := &Circuit{
		log:           l,
		maxCurrent:    limit,
		ParentCircuit: p,
	}
	return circuit
}

// NewCircuitFromConfig creates circuit from config
func NewCircuitFromConfig(log *util.Logger, cp configProvider, other map[string]interface{}) (cc *Circuit, name string, parentName string, e error) {
	var circuitCfg = new(CircuitConfig)

	if err := util.DecodeOther(other, circuitCfg); err != nil {
		return nil, "", "", err
	}
	if circuitCfg.Name == "" {
		return nil, "", "", fmt.Errorf("circuit name must not be empty")
	}

	var parentCircuit *Circuit
	var err error
	if circuitCfg.ParentRef != "" {
		parentCircuit, err = cp.Circuit(circuitCfg.ParentRef)
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to set parent circuit %s: %w", circuitCfg.ParentRef, err)
		}
	}

	cc = NewCircuit(circuitCfg.MaxCurrent, parentCircuit, log)
	var phaseCurrents *api.PhaseCurrents
	if circuitCfg.MeterRef != "" {
		mt, err := cp.Meter(circuitCfg.MeterRef)
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to set meter %s: %w", circuitCfg.MeterRef, err)
		}
		if pm, ok := mt.(api.PhaseCurrents); ok {
			phaseCurrents = &pm
		} else {
			return nil, "", "", fmt.Errorf("circuit needs meter with phase current support: %s", circuitCfg.MeterRef)
		}
		cc.PhaseCurrents = *phaseCurrents
	}

	return cc, circuitCfg.Name, circuitCfg.ParentRef, nil
}

// DumpConfig dumps the current circuit
func (circuit *Circuit) DumpConfig(indent int, maxIndent int) string {

	var parentLimit float64
	if circuit.ParentCircuit != nil {
		parentLimit = circuit.ParentCircuit.maxCurrent
	}
	return fmt.Sprintf("%s maxCurrent %.1fA (parent: %.1fA)",
		strings.Repeat(" ", indent),
		circuit.maxCurrent,
		parentLimit)
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
	circuit.publish("maxCurrent", circuit.maxCurrent)
}

// update gets called on every site update call.
// this is used to update the current consumption etc to get published in status and databases
func (circuit *Circuit) update() {
	_, _ = circuit.MaxPhasesCurrent()
}
