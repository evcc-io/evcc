package core

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

var gridMeterUsed bool // indicates gridmeter is used already fo a circuit to avoid > 1 usage

type Circuit struct {
	log    *util.Logger
	uiChan chan<- util.Param

	Name       string     `mapstructure:"name"`       // meaningful name, used as reference in lp
	MaxCurrent float64    `mapstructure:"maxCurrent"` // the max allowed current of this circuit
	MeterRef   string     `mapstructure:"meter"`      // Charge meter reference
	Circuits   []*Circuit `mapstructure:"circuits"`   // sub circuits as config reference

	parentCircuit *Circuit          // parent circuit reference
	phaseMeter    api.PhaseCurrents // meter to determine phase current
	vMeter        *VMeter           // virtual meter if no real meter is used
}

// GetCurrent determines current in use. Implements consumer interface
// TBD: phase perfect evaluation
func (circuit *Circuit) MaxPhasesCurrent() (float64, error) {
	var current float64

	i1, i2, i3, err := circuit.phaseMeter.Currents()
	if err != nil {
		return 0, fmt.Errorf("failed getting meter currents: %w", err)
	}
	circuit.log.DEBUG.Printf("meter currents: %.3gA", []float64{i1, i2, i3})
	circuit.publish("meterCurrents", []float64{i1, i2, i3})
	// TODO: phase adjusted handling. Currently we take highest current from all phases
	current = math.Max(math.Max(i1, i2), i3)

	circuit.log.DEBUG.Printf("actual current: %.1fA", current)
	circuit.publish("actualCurrent", current)
	return current, nil
}

// GetRemainingCurrent avaialble current based on limit and consumption
// checks down up to top level parent
func (circuit *Circuit) GetRemainingCurrent() float64 {
	circuit.log.TRACE.Printf("get available current")
	// first update current current, mainly to regularly publish the value
	current, err := circuit.MaxPhasesCurrent()
	if err != nil {
		circuit.log.ERROR.Printf("max phase currents: %v", err)
		return 0
	}
	curAvailable := circuit.MaxCurrent - current
	if curAvailable < 0.0 {
		circuit.log.WARN.Printf("overload detected (%s) - currents: %.1fA, allowed max current is: %.1fA\n", circuit.Name, current, circuit.MaxCurrent)
		circuit.publish("overload", true)
	} else {
		circuit.publish("overload", false)
	}
	// check parent circuit, return lowest
	if circuit.parentCircuit != nil {
		circuit.log.TRACE.Printf("get available current from parent: %s", circuit.parentCircuit.Name)
		curAvailable = math.Min(curAvailable, circuit.parentCircuit.GetRemainingCurrent())
	}
	circuit.log.DEBUG.Printf("circuit using %.1fA, %.1fA available", current, curAvailable)
	return curAvailable
}

// NewCircuit a circuit with defaults
func NewCircuit(n string, limit float64, mc api.PhaseCurrents, l *util.Logger) *Circuit {
	circuit := &Circuit{
		Name:       n,
		log:        l,
		MaxCurrent: limit,
		phaseMeter: mc,
	}
	return circuit
}

// NewCircuitFromConfig creates circuit from config
// using site to get access to the grid meter if configured, see cp.Meter() for details
func NewCircuitFromConfig(cp configProvider, other map[string]interface{}, site *Site) (*Circuit, error) {
	var circuit = new(Circuit)
	if err := util.DecodeOther(other, circuit); err != nil {
		return nil, err
	}

	circuit.log = util.NewLogger("circuit-" + circuit.Name)
	circuit.log.TRACE.Println("NewCircuitFromConfig()")
	circuit.PrintCircuits(0) // for tracing only
	if err := circuit.InitCircuits(site, cp); err != nil {
		return nil, err
	}
	circuit.PrintCircuits(0) // for tracing only
	circuit.log.TRACE.Printf("created new circuit: %s, limit: %.1fA", circuit.Name, circuit.MaxCurrent)
	circuit.log.TRACE.Println("NewCircuitFromConfig()) end")
	return circuit, nil
}

// InitCircuits initializes circuits in hierarchy incl meters
func (circuit *Circuit) InitCircuits(site *Site, cp configProvider) error {
	if circuit.Name == "" {
		return fmt.Errorf("circuit name must not be empty")
	}

	circuit.log = util.NewLogger("circuit-" + circuit.Name)
	circuit.log.TRACE.Printf("InitCircuits(): %s (%p)", circuit.Name, circuit)
	if circuit.MeterRef != "" {
		var (
			mt  api.Meter
			err error
		)
		if circuit.MeterRef == site.Meters.GridMeterRef {
			if gridMeterUsed {
				return fmt.Errorf("grid meter used more in more than one circuit: %s", circuit.MeterRef)
			}
			mt = site.gridMeter
			gridMeterUsed = true
			circuit.log.TRACE.Printf("add grid meter from site: %s", circuit.MeterRef)
		} else {
			mt, err = cp.Meter(circuit.MeterRef)
			if err != nil {
				return fmt.Errorf("failed to set meter %s: %w", circuit.MeterRef, err)
			}
			circuit.log.TRACE.Printf("add separate meter: %s", circuit.MeterRef)
		}
		if pm, ok := mt.(api.PhaseCurrents); ok {
			circuit.phaseMeter = pm
		} else {
			return fmt.Errorf("circuit needs meter with phase current support: %s", circuit.MeterRef)
		}
	} else {
		// create virtual meter
		circuit.vMeter = NewVMeter(circuit.Name)
		circuit.phaseMeter = circuit.vMeter
	}
	// initialize also included circuits
	if circuit.Circuits != nil {
		for ccId := range circuit.Circuits {
			circuit.log.TRACE.Printf("creating circuit from circuitRef: %s", circuit.Circuits[ccId].Name)
			circuit.Circuits[ccId].parentCircuit = circuit
			if err := circuit.Circuits[ccId].InitCircuits(site, cp); err != nil {
				return err
			}
			if vmtr := circuit.GetVMeter(); vmtr != nil {
				vmtr.AddConsumer(circuit.Circuits[ccId])
			}
			circuit.Circuits[ccId].PrintCircuits(0)
		}
	} else {
		circuit.log.TRACE.Printf("no sub circuits")
	}
	circuit.log.TRACE.Println("InitCircuits() exit")
	circuit.log.INFO.Printf("initialized new circuit: %s, limit: %.1fA", circuit.Name, circuit.MaxCurrent)
	return nil
}

// GetVMeter returns the meter used in circuit
func (circuit *Circuit) GetVMeter() *VMeter {
	return circuit.vMeter
}

// PrintCircuits dumps recursively circuit config
// trace output of circuit and subcircuits
func (circuit *Circuit) PrintCircuits(indent int) {
	for _, s := range circuit.DumpConfig(0, 15) {
		circuit.log.TRACE.Println(s)
	}
}

// DumpConfig dumps the current circuit
// returns string array to dump the config
func (circuit *Circuit) DumpConfig(indent int, maxIndent int) []string {

	var res []string

	cfgDump := fmt.Sprintf("%s%s:%s meter %s maxCurrent %.1fA",
		strings.Repeat(" ", indent),
		circuit.Name,
		strings.Repeat(" ", maxIndent-len(circuit.Name)-indent),
		presence[circuit.GetVMeter() == nil],
		circuit.MaxCurrent,
	)
	res = append(res, cfgDump)

	// cc.Log.TRACE.Printf("%s%s%s: (%p) log: %t, meter: %t, parent: %p\n", strings.Repeat(" ", indent), cc.Name, strings.Repeat(" ", 10-indent), cc, cc.Log != nil, cc.meterCurrent != nil, cc.parentCircuit)
	for _, subCircuit := range circuit.Circuits {
		// this does not work (compiler error), but linter requests it. Github wont build ...
		// res = append(res, cc.Circuits[id].DumpConfig(indent+2, maxIndent))
		// hacky work around
		for _, l := range subCircuit.DumpConfig(indent+2, maxIndent) {
			res = append(res, l)
			// add useless command
			time.Sleep(0)
		}
	}
	return res
}

// GetCircuit returns the circiut with given name, checking all subcircuits
func (circuit *Circuit) GetCircuit(n string) *Circuit {
	circuit.log.TRACE.Printf("searching for circuit %s in %s", n, circuit.Name)
	if circuit.Name == n {
		circuit.log.TRACE.Printf("found circuit %s (%p)", circuit.Name, &circuit)
		return circuit
	} else {
		for _, subCircuit := range circuit.Circuits {
			circuit.log.TRACE.Printf("start looking in circuit %s (%p)", subCircuit.Name, &subCircuit)
			retCC := subCircuit.GetCircuit(n)
			if retCC != nil {
				circuit.log.TRACE.Printf("found circuit %s (%p)", retCC.Name, &retCC)
				return retCC
			}
		}
	}
	circuit.log.INFO.Printf("could not find circuit %s", n)
	return nil
}

// publish sends values to UI and databases
func (circuit *Circuit) publish(key string, val interface{}) {
	// test helper
	if circuit.uiChan == nil {
		return
	}

	circuit.uiChan <- util.Param{
		Circuit: &circuit.Name,
		Key:     key,
		Val:     val,
	}
}

// Prepare set the UI channel to publish information
func (circuit *Circuit) Prepare(uiChan chan<- util.Param) {
	circuit.uiChan = uiChan
	circuit.publish("name", circuit.Name)
	circuit.publish("maxCurrent", circuit.MaxCurrent)
	if vmtr := circuit.GetVMeter(); vmtr != nil {
		circuit.publish("virtualMeter", true)
		circuit.publish("consumers", len(vmtr.Consumers)-len(circuit.Circuits))
	} else {
		circuit.publish("virtualMeter", false)
	}
	// initialize sub circuits
	for _, subCircuit := range circuit.Circuits {
		subCircuit.Prepare(uiChan)
	}
}

// update gets called on every site update call.
// this is used to update the current consumption etc to get published in status and databases
func (circuit *Circuit) update() {
	_, _ = circuit.MaxPhasesCurrent()
	for _, subCircuit := range circuit.Circuits {
		subCircuit.update()
	}
}
