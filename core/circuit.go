package core

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

var _ circuit.API = (*Circuit)(nil)

// the circuit instances to control the load
type Circuit struct {
	log    *util.Logger
	uiChan chan<- util.Param

	children []*Circuit // parent circuit reference, used to determine current maxs from hierarchy
	parent   *Circuit   // parent circuit reference, used to determine current maxs from hierarchy
	meter    api.Meter  // meter to determine current power

	maxCurrent float64 // max allowed current
	maxPower   float64 // max allowed power

	current float64
	power   float64
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
	c := &Circuit{
		log:        log,
		maxCurrent: maxCurrent,
		maxPower:   maxPower,
		meter:      meter,
	}

	if maxCurrent == 0 {
		c.log.DEBUG.Printf("validation of max phase current disabled")
	} else if _, ok := meter.(api.PhaseCurrents); !ok {
		return nil, fmt.Errorf("meter does not support phase currents")
	}

	if maxPower == 0 {
		c.log.DEBUG.Printf("validation of max power disabled")
	}

	return c, nil
}

func (c *Circuit) WithParent(parent *Circuit) {
	c.parent = parent
}

func (c *Circuit) updateLoadpoints(loadpoints []loadpoint.API) {
	var total float64
	for _, lp := range loadpoints {
		if lp.GetCircuit() == c {
			total += lp.GetChargePower()
		}
		// TODO currents
	}
	c.power = total
}

func (c *Circuit) Update(loadpoints []loadpoint.API) error {
	// TODO retry
	if c.meter != nil {
		if f, err := c.meter.CurrentPower(); err == nil {
			c.power = f
		} else {
			return fmt.Errorf("circuit power: %w", err)
		}

		if phaseMeter, ok := c.meter.(api.PhaseCurrents); ok {
			if l1, l2, l3, err := phaseMeter.Currents(); err == nil {
				// TODO handle negative currents
				c.current = max(l1, l2, l3)
			} else {
				return fmt.Errorf("circuit currents: %w", err)
			}
		}

		return nil
	}

	c.updateLoadpoints(loadpoints)

	for _, ch := range c.children {
		if err := ch.Update(loadpoints); err != nil {
			return err
		}

		c.power += ch.GetChargePower()
		c.current += ch.GetChargeCurrent()
	}

	return nil
}

// GetChargePower returns the actual power
func (c *Circuit) GetChargePower() float64 {
	return c.power
}

// GetChargeCurrent returns the actual current
func (c *Circuit) GetChargeCurrent() float64 {
	return c.current
}

// ValidateCurrent returns the actual current
func (c *Circuit) ValidateCurrent(old, new float64) float64 {
	delta := max(0, new-old)

	if c.current+delta <= c.maxCurrent {
		return new
	}

	return max(0, c.maxCurrent-c.current)
}

// // publish sends values to UI and databases
// func (c *Circuit) publish(key string, val interface{}) {
// 	// test helper
// 	if c.uiChan != nil {
// 		c.uiChan <- util.Param{Key: key, Val: val}
// 	}
// }

// // Prepare set the UI channel to publish information
// func (c *Circuit) Prepare(uiChan chan<- util.Param) {
// 	c.uiChan = uiChan
// 	if c.maxCurrent != math.MaxFloat64 {
// 		c.publish("maxCurrent", c.maxCurrent)
// 	}
// 	if c.maxPower != math.MaxFloat64 {
// 		c.publish("maxPower", c.maxPower)
// 	}
// }

// // update gets called on every site update call.
// // this is used to update the current consumption etc to get published in status and databases
// func (c *Circuit) update() error {
// 	if c.phaseCurrents != nil {
// 		if _, err := c.MaxPhasesCurrent(); err != nil {
// 			return err
// 		}
// 	}
// 	_, err := c.CurrentPower()
// 	return err
// }

// var _ Consumer = (*Circuit)(nil)

// // CurrentPower implements consumer interface and determines actual power in use.
// func (c *Circuit) CurrentPower() (float64, error) {
// 	return c.powerMeter.CurrentPower()
// }

// // GetRemainingPower determines the power left to be used from configured maxPower
// func (c *Circuit) GetRemainingPower() float64 {
// 	power, err := c.CurrentPower()
// 	if err != nil {
// 		c.log.ERROR.Printf("power currents: %v", err)
// 		return 0
// 	}

// 	if c.maxPower == math.MaxFloat64 && c.parentCircuit == nil {
// 		return c.maxPower
// 	}

// 	powerAvailable := c.maxPower - power
// 	c.publish("overload", powerAvailable < 0)
// 	if powerAvailable < 0 {
// 		c.log.WARN.Printf("overload detected - power: %.2fkW, allowed max power is: %.2fkW\n", power/1000, c.maxPower/1000)
// 	}

// 	// check parent circuit, return lowest
// 	if c.parentCircuit != nil {
// 		powerAvailable = math.Min(powerAvailable, c.parentC.GetRemainingPower())
// 	}

// 	if powerAvailable/1000 > 10000.0 {
// 		c.log.DEBUG.Printf("circuit using %.2fkW, no max checking", power/1000)
// 	} else {
// 		c.log.DEBUG.Printf("circuit using %.2fkW, %.2fkW available", power/1000, powerAvailable/1000)
// 	}

// 	return powerAvailable
// }

// // MaxPhasesCurrent determines current in use. Implements consumer interface
// func (c *Circuit) MaxPhasesCurrent() (float64, error) {
// 	if c.phaseCurrents == nil {
// 		return 0, fmt.Errorf("no phase meter assigned")
// 	}
// 	i1, i2, i3, err := c.phaseCurrents.Currents()
// 	if err != nil {
// 		return 0, fmt.Errorf("failed getting meter currents: %w", err)
// 	}

// 	c.log.TRACE.Printf("meter currents: %.3gA", []float64{i1, i2, i3})
// 	c.publish("meterCurrents", []float64{i1, i2, i3})

// 	// TODO: phase adjusted handling. Currently we take highest current from all phases
// 	current := math.Max(math.Max(i1, i2), i3)

// 	c.log.TRACE.Printf("actual current: %.1fA", current)
// 	c.publish("actualCurrent", current)

// 	return current, nil
// }

// // GetRemainingCurrent available current based on max and consumption
// // checks down up to top level parent
// func (c *Circuit) GetRemainingCurrent() float64 {
// 	if c.maxCurrent == math.MaxFloat64 && c.parentCircuit == nil {
// 		return c.maxCurrent
// 	}

// 	current, err := c.MaxPhasesCurrent()
// 	if err != nil {
// 		c.log.ERROR.Printf("max phase currents: %v", err)
// 		return 0
// 	}

// 	curAvailable := c.maxCurrent - current
// 	c.publish("overload", curAvailable < 0)
// 	if curAvailable < 0 {
// 		c.log.WARN.Printf("overload detected - currents: %.1fA, allowed max current is: %.1fA\n", current, c.maxCurrent)
// 	}

// 	// check parent circuit, return lowest
// 	if c.parentCircuit != nil {
// 		curAvailable = math.Min(curAvailable, c.parentC.GetRemainingCurrent())
// 	}
// 	if curAvailable > 10000.0 {
// 		c.log.DEBUG.Printf("circuit using %.1fA, no max checking", current)
// 	} else {
// 		c.log.DEBUG.Printf("circuit using %.1fA, %.1fA available", current, curAvailable)
// 	}
// 	return curAvailable
// }
