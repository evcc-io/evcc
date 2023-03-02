package core

import (
	"fmt"
	"math"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

var (
	circuitId int // counter for circuit id
)

// struct to setup the circuit hierarchy
type CircuitConfig struct {
	// Title      string           `mapstructure:"title"`      // printable name
	Name       string           `mapstructure:"name"`       // unique name, used as reference in lp
	MaxCurrent float64          `mapstructure:"maxCurrent"` // the max allowed current of this circuit
	MeterRef   string           `mapstructure:"meter"`      // Charge meter reference
	Circuits   []*CircuitConfig `mapstructure:"circuits"`   // sub circuits as config reference

	CircuitRef *Circuit // reference to instance
	vMeter     *VMeter  // virtual meter for the circuit, if needed
}

// the circuit instances to control the load
type Circuit struct {
	log    *util.Logger
	uiChan chan<- util.Param

	Title         string            // pretty logging
	maxCurrent    float64           // max allowed current
	parentCircuit *Circuit          // parent circuit reference, used to determine current limits from hierarchy
	phaseMeter    api.PhaseCurrents // meter to determine phase current
}

// GetCurrent determines current in use. Implements consumer interface
// TBD: phase perfect evaluation
func (circuit *Circuit) MaxPhasesCurrent() (float64, error) {
	var current float64

	i1, i2, i3, err := circuit.phaseMeter.Currents()
	if err != nil {
		return 0, fmt.Errorf("failed getting meter currents: %w", err)
	}
	circuit.log.DEBUG.Printf("(%s) meter currents: %.3gA", circuit.Title, []float64{i1, i2, i3})
	circuit.publish("meterCurrents", []float64{i1, i2, i3})
	// TODO: phase adjusted handling. Currently we take highest current from all phases
	current = math.Max(math.Max(i1, i2), i3)

	circuit.log.DEBUG.Printf("(%s) actual current: %.1fA", circuit.Title, current)
	circuit.publish("actualCurrent", current)
	return current, nil
}

// GetRemainingCurrent avaialble current based on limit and consumption
// checks down up to top level parent
func (circuit *Circuit) GetRemainingCurrent() float64 {
	circuit.log.TRACE.Printf("(%s) get available current", circuit.Title)
	// first update current current, mainly to regularly publish the value
	current, err := circuit.MaxPhasesCurrent()
	if err != nil {
		circuit.log.ERROR.Printf("(%s) max phase currents: %v", circuit.Title, err)
		return 0
	}
	curAvailable := circuit.maxCurrent - current
	if curAvailable < 0.0 {
		circuit.log.WARN.Printf("(%s) overload detected (%s) - currents: %.1fA, allowed max current is: %.1fA\n", circuit.Title, circuit.Title, current, circuit.maxCurrent)
		circuit.publish("overload", true)
	} else {
		circuit.publish("overload", false)
	}
	// check parent circuit, return lowest
	if circuit.parentCircuit != nil {
		circuit.log.TRACE.Printf("(%s) get available current from parent: %s", circuit.Title, circuit.parentCircuit.Title)
		curAvailable = math.Min(curAvailable, circuit.parentCircuit.GetRemainingCurrent())
	}
	circuit.log.DEBUG.Printf("(%s) circuit using %.1fA, %.1fA available", circuit.Title, current, curAvailable)
	return curAvailable
}

// NewCircuit a circuit with defaults
func NewCircuit(t string, limit float64, pm api.PhaseCurrents, l *util.Logger) *Circuit {
	circuit := &Circuit{
		log:        l,
		Title:      t,
		maxCurrent: limit,
		phaseMeter: pm,
	}
	circuitId += 1
	return circuit
}

// NewCircuitFromConfig creates circuit from config
// using site to get access to the grid meter if configured, see cp.Meter() for details
// returns a map of circuit name and circuit ref
func NewCircuitFromConfig(cp configProvider, other map[string]interface{}) (map[string]*Circuit, map[string]*VMeter, error) {
	var circuitCfg = new(CircuitConfig)

	if err := util.DecodeOther(other, circuitCfg); err != nil {
		return nil, nil, err
	}
	if circuitCfg.Name == "" {
		return nil, nil, fmt.Errorf("circuit name must not be empty")
	}

	// collect circuits and vmeters per circuit name as return for setup
	circuitMap := map[string]*Circuit{}
	vmeterMap := map[string]*VMeter{}

	var circuit = NewCircuit(circuitCfg.Name, circuitCfg.MaxCurrent, nil, util.NewLogger(fmt.Sprintf("circuit-%d", circuitId)))
	// remember this instance in config
	circuitCfg.CircuitRef = circuit
	circuitMap[circuitCfg.Name] = circuit

	// append for result
	circuitMap[circuitCfg.Name] = circuit

	circuit.log.TRACE.Printf("(%s) NewCircuitFromConfig()", circuit.Title)
	circuit.PrintCircuits(0) // for tracing only
	if err := circuit.InitCircuits(circuitCfg, cp, circuitMap, vmeterMap); err != nil {
		return nil, nil, err
	}
	circuit.PrintCircuits(0) // for tracing only
	circuit.log.TRACE.Printf("created new circuit: %s, limit: %.1fA", circuit.Title, circuit.maxCurrent)
	circuit.log.TRACE.Println("NewCircuitFromConfig()) end")

	// build the map / circuit
	return circuitMap, vmeterMap, nil
}

// InitCircuits initializes circuits in hierarchy incl meters
func (circuit *Circuit) InitCircuits(circuitCfg *CircuitConfig, cp configProvider, circuitMap map[string]*Circuit, vmeterMap map[string]*VMeter) error {
	if circuitCfg.Name == "" {
		return fmt.Errorf("circuit name must not be empty")
	}

	circuit.log.TRACE.Printf("(%s) InitCircuits(): %p", circuit.Title, circuit)
	if circuitCfg.MeterRef != "" {
		// use confiured meter
		circuit.log.TRACE.Printf("(%s) add separate meter: %s", circuit.Title, circuitCfg.MeterRef)
		mt, err := cp.Meter(circuitCfg.MeterRef)
		if err != nil {
			return fmt.Errorf("failed to set meter %s: %w", circuitCfg.MeterRef, err)
		}
		if pm, ok := mt.(api.PhaseCurrents); ok {
			circuit.phaseMeter = pm
		} else {
			return fmt.Errorf("circuit needs meter with phase current support: %s", circuitCfg.MeterRef)
		}
	} else {
		// create virtual meter
		circuit.log.DEBUG.Printf("(%s) no meter configured, create virtual meter", circuit.Title)
		circuitCfg.vMeter = NewVMeter(circuit.Title)
		circuit.phaseMeter = circuitCfg.vMeter
		vmeterMap[circuitCfg.Name] = circuitCfg.vMeter
	}
	// initialize also included circuits
	if circuitCfg.Circuits != nil {
		for ccId := range circuitCfg.Circuits {
			var subCircuitCfg = circuitCfg.Circuits[ccId]
			circuit.log.TRACE.Printf("(%s) creating circuit from circuitRef: %s", circuitCfg.Name, subCircuitCfg.Name)

			// check name not alredy exists
			if _, ok := circuitMap[subCircuitCfg.Name]; ok {
				return fmt.Errorf("circuit name alredy in use: %s", circuitCfg.Name)
			}

			// the new circuit instance
			var subCircuit = NewCircuit(subCircuitCfg.Name, subCircuitCfg.MaxCurrent, nil, util.NewLogger(fmt.Sprintf("circuit-%d", circuitId)))
			circuitMap[subCircuitCfg.Name] = subCircuit
			subCircuitCfg.CircuitRef = subCircuit

			// remember this instance in config
			subCircuitCfg.CircuitRef = circuit

			subCircuit.parentCircuit = circuit
			if err := subCircuit.InitCircuits(subCircuitCfg, cp, circuitMap, vmeterMap); err != nil {
				return err
			}
			// if this has vMeter, add subcircuit to consumers
			if circuitCfg.vMeter != nil {
				circuitCfg.vMeter.AddConsumer(subCircuit)
			}
			subCircuit.PrintCircuits(0)
		}
	} else {
		circuit.log.TRACE.Printf("(%s) no sub circuits", circuit.Title)
	}
	circuit.log.TRACE.Printf("(%s) InitCircuits() exit", circuit.Title)
	circuit.log.INFO.Printf("(%s) initialized new circuit. Limit: %.1fA", circuit.Title, circuit.maxCurrent)
	return nil
}

// PrintCircuits dumps circuit config
func (circuit *Circuit) PrintCircuits(indent int) {
	circuit.log.TRACE.Println(circuit.DumpConfig(indent, 15))
}

// DumpConfig dumps the current circuit
// returns string array to dump the config
func (circuit *Circuit) DumpConfig(indent int, maxIndent int) string {

	parentTitle := ""
	if circuit.parentCircuit != nil {
		parentTitle = fmt.Sprintf(" (member of %s)", circuit.parentCircuit.Title)
	}

	titleLen := len(circuit.Title)
	if titleLen > maxIndent-indent {
		titleLen = maxIndent - indent
	}

	return fmt.Sprintf("%s%s:%s maxCurrent %.1fA%s",
		strings.Repeat(" ", indent),
		circuit.Title[0:titleLen],
		strings.Repeat(" ", maxIndent-titleLen-indent),
		circuit.maxCurrent,
		parentTitle)

}

// publish sends values to UI and databases
func (circuit *Circuit) publish(key string, val interface{}) {
	// test helper
	if circuit.uiChan == nil {
		return
	}

	circuit.uiChan <- util.Param{
		Circuit: &circuit.Title,
		Key:     key,
		Val:     val,
	}
}

// Prepare set the UI channel to publish information
func (circuit *Circuit) Prepare(uiChan chan<- util.Param) {
	circuit.uiChan = uiChan
	circuit.publish("title", circuit.Title)
	circuit.publish("maxCurrent", circuit.maxCurrent)
}

// update gets called on every site update call.
// this is used to update the current consumption etc to get published in status and databases
func (circuit *Circuit) update() {
	_, _ = circuit.MaxPhasesCurrent()
}
