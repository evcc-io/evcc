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
	log *util.Logger
	// uiChan chan<- util.Param

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
	var totalPower, totalCurrent float64

	for _, lp := range loadpoints {
		if lp.GetCircuit() != c {
			continue
		}

		totalPower += lp.GetChargePower()
		totalCurrent += max(lp.GetChargeCurrents())
	}

	c.power = totalPower
	c.current = totalCurrent
}

func (c *Circuit) GetParent() *Circuit {
	// if reflect.ValueOf(c.parent).IsNil() {
	// 	return nil
	// }
	return c.parent
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

	if c.maxCurrent == 0 || c.current+delta <= c.maxCurrent {
		return new
	}

	return max(0, c.maxCurrent-c.current)
}

// ValidatePower returns the actual power
func (c *Circuit) ValidatePower(old, new float64) float64 {
	delta := max(0, new-old)

	if c.maxPower == 0 || c.power+delta <= c.maxPower {
		return new
	}

	return max(0, c.maxPower-c.power)
}
