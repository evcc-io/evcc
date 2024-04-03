package core

import (
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

var _ api.Circuit = (*Circuit)(nil)

// the circuit instances to control the load
type Circuit struct {
	mu  sync.RWMutex
	log *util.Logger

	title    string
	parent   api.Circuit   // parent circuit
	children []api.Circuit // child circuits
	meter    api.Meter     // meter to determine current power

	maxCurrent float64 // max allowed current
	maxPower   float64 // max allowed power

	current float64
	power   float64
}

// NewCircuitFromConfig creates a new Circuit
func NewCircuitFromConfig(log *util.Logger, other map[string]interface{}) (api.Circuit, error) {
	var cc struct {
		Title      string  `mapstructure:"title"`      // title
		ParentRef  string  `mapstructure:"parent"`     // parent circuit reference
		MeterRef   string  `mapstructure:"meter"`      // meter reference
		MaxCurrent float64 `mapstructure:"maxCurrent"` // the max allowed current
		MaxPower   float64 `mapstructure:"maxPower"`   // the max allowed power
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

	circuit, err := NewCircuit(log, cc.Title, cc.MaxCurrent, cc.MaxPower, meter)
	if err != nil {
		return nil, err
	}

	if cc.ParentRef != "" {
		dev, err := config.Circuits().ByName(cc.ParentRef)
		if err != nil {
			return nil, err
		}
		circuit.SetParent(dev.Instance())
	}

	return circuit, err
}

// NewCircuit creates a circuit
func NewCircuit(log *util.Logger, title string, maxCurrent, maxPower float64, meter api.Meter) (*Circuit, error) {
	c := &Circuit{
		log:        log,
		title:      title,
		maxCurrent: maxCurrent,
		maxPower:   maxPower,
		meter:      meter,
	}

	if maxPower == 0 {
		c.log.DEBUG.Printf("validation of max power disabled")
	}

	if maxCurrent == 0 {
		c.log.DEBUG.Printf("validation of max phase current disabled")
	} else if _, ok := meter.(api.PhaseCurrents); meter != nil && !ok {
		return nil, fmt.Errorf("meter does not support phase currents")
	}

	return c, nil
}

func (c *Circuit) GetTitle() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.title
}

func (c *Circuit) SetTitle(title string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.title = title
}

// GetParent returns the parent circuit
func (c *Circuit) GetParent() api.Circuit {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.parent
}

// SetParent set parent circuit
func (c *Circuit) SetParent(parent api.Circuit) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.parent = parent
	if parent != nil {
		parent.RegisterChild(c)
	}
}

// HasMeter returns the max power setting
func (c *Circuit) HasMeter() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.meter != nil
}

// GetMaxPower returns the max power setting
func (c *Circuit) GetMaxPower() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxPower
}

// GetMaxCurrent returns the max current setting
func (c *Circuit) GetMaxCurrent() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxCurrent
}

// RegisterChild registers child circuit
func (c *Circuit) RegisterChild(child api.Circuit) {
	c.children = append(c.children, child)
}

func (c *Circuit) updateLoadpoints(loadpoints []api.CircuitLoad) {
	c.power = 0
	c.current = 0

	for _, lp := range loadpoints {
		if lp.GetCircuit() != c {
			continue
		}

		c.power += lp.GetChargePower()
		c.current += lp.GetMaxPhaseCurrent()
	}
}

func (c *Circuit) updateMeters() error {
	if f, err := c.meter.CurrentPower(); err == nil {
		// TODO handle negative powers
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

func (c *Circuit) Update(loadpoints []api.CircuitLoad) (err error) {
	defer func() {
		if c.maxPower != 0 && c.power > c.maxPower {
			c.log.WARN.Printf("over power detected: %gW > %gW", c.power, c.maxPower)
		} else {
			c.log.DEBUG.Printf("power: %gW", c.power)
		}

		if c.maxCurrent != 0 && c.current > c.maxCurrent {
			c.log.WARN.Printf("over current detected: %gA > %gA", c.current, c.maxCurrent)
		} else {
			c.log.DEBUG.Printf("current: %gA", c.current)
		}
	}()

	// update children depth-first
	for _, ch := range c.children {
		if err := ch.Update(loadpoints); err != nil {
			return err
		}
	}

	// meter available
	if c.meter != nil {
		return c.updateMeters()
	}

	// no meter available
	c.updateLoadpoints(loadpoints)
	for _, ch := range c.children {
		c.power += ch.GetChargePower()
		c.current += ch.GetMaxPhaseCurrent()
	}

	return nil
}

// GetChargePower returns the actual power
func (c *Circuit) GetChargePower() float64 {
	return c.power
}

// GetMaxPhaseCurrent returns the actual current
func (c *Circuit) GetMaxPhaseCurrent() float64 {
	return c.current
}

// ValidatePower validates power request
func (c *Circuit) ValidatePower(old, new float64) float64 {
	delta := max(0, new-old)

	if c.maxPower != 0 {
		if c.power+delta > c.maxPower {
			new = max(0, c.maxPower-c.power)
			c.log.DEBUG.Printf("validate power: %gW -> %gW <= %gW at %gW: capped at %gW", old, new, c.maxPower, c.power, new)
		} else {
			c.log.TRACE.Printf("validate power: %gW -> %gW <= %gW at %gW: ok", old, new, c.maxPower, c.power)
		}
	}

	if c.parent != nil {
		res := c.parent.ValidatePower(c.power, new)
		if res != new {
			c.log.TRACE.Printf("validate power: %gW -> %gW at %gW: capped at %gW", old, new, c.power, new)
		}
		return res
	}

	return new
}

// ValidateCurrent validates current request
func (c *Circuit) ValidateCurrent(old, new float64) (res float64) {
	delta := max(0, new-old)

	if c.maxCurrent != 0 {
		if c.current+delta > c.maxCurrent {
			new = max(0, c.maxCurrent-c.current)
			c.log.DEBUG.Printf("validate current: %gA -> %gA <= %gA at %gA: capped at %gA", old, new, c.maxCurrent, c.current, new)
		} else {
			c.log.TRACE.Printf("validate current: %gA -> %gA <= %gA at %gA: ok", old, new, c.maxCurrent, c.current)
		}
	}

	if c.parent != nil {
		res := c.parent.ValidateCurrent(c.current, new)
		if res != new {
			c.log.TRACE.Printf("validate current: %gA -> %gA at %gA: capped by parent at %gA", old, new, c.current, res)
		}
		return res
	}

	return new
}

// func (c *Circuit) validate(typ string, current, old, new float64, parentFunc func(o, n float64) float64) float64 {
// 	delta := max(0, new-old)

// 	if c.maxPower != 0 {
// 		if c.power+delta > c.maxPower {
// 			new = max(0, c.maxPower-c.power)
// 			c.log.TRACE.Printf("validate power: %g -> %g <= %g at %g: capped at %g", old, new, c.maxPower, c.power, new)
// 		} else {
// 			c.log.TRACE.Printf("validate power: %g -> %g <= %g at %g: ok", old, new, c.maxPower, c.power)
// 		}
// 	}

// 	if c.parent != nil {
// 		return c.parent.ValidatePower(c.power, new)
// 	}

// 	return new
// }
