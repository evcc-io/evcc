package core

import (
	"fmt"
	"math"
	"sync"
	"time"

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
	timeout  time.Duration

	maxCurrent float64 // max allowed current
	maxPower   float64 // max allowed power

	current float64
	power   float64

	currentUpdated time.Time
	powerUpdated   time.Time
}

// NewCircuitFromConfig creates a new Circuit
func NewCircuitFromConfig(log *util.Logger, other map[string]interface{}) (api.Circuit, error) {
	cc := struct {
		Title      string        `mapstructure:"title"`      // title
		ParentRef  string        `mapstructure:"parent"`     // parent circuit reference
		MeterRef   string        `mapstructure:"meter"`      // meter reference
		MaxCurrent float64       `mapstructure:"maxCurrent"` // the max allowed current
		MaxPower   float64       `mapstructure:"maxPower"`   // the max allowed power
		Timeout    time.Duration `mapstructure:"timeout"`    // timeout between meter updates
	}{
		Timeout: time.Minute,
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

	circuit, err := NewCircuit(log, cc.Title, cc.MaxCurrent, cc.MaxPower, meter, cc.Timeout)
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
func NewCircuit(log *util.Logger, title string, maxCurrent, maxPower float64, meter api.Meter, timeout time.Duration) (*Circuit, error) {
	c := &Circuit{
		log:        log,
		title:      title,
		maxCurrent: maxCurrent,
		maxPower:   maxPower,
		meter:      meter,
		timeout:    timeout,
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

// SetMaxPower sets the max power
func (c *Circuit) SetMaxPower(power float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxPower = power
}

// GetMaxCurrent returns the max current setting
func (c *Circuit) GetMaxCurrent() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxCurrent
}

// SetMaxCurrent sets the max current
func (c *Circuit) SetMaxCurrent(current float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxCurrent = current
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

func (c *Circuit) overloadOnError(t time.Time, val *float64) {
	if c.timeout > 0 && time.Since(t) > c.timeout {
		*val = math.MaxFloat64
	}
}

func (c *Circuit) updateMeters() error {
	if f, err := c.meter.CurrentPower(); err == nil {
		c.power = f
		c.powerUpdated = time.Now()
	} else {
		c.overloadOnError(c.powerUpdated, &c.power)
		return fmt.Errorf("circuit power: %w", err)
	}

	if phaseMeter, ok := c.meter.(api.PhaseCurrents); ok {
		if l1, l2, l3, err := phaseMeter.Currents(); err == nil {
			c.current = max(l1, l2, l3)
			c.currentUpdated = time.Now()
		} else {
			c.overloadOnError(c.currentUpdated, &c.current)
			return fmt.Errorf("circuit currents: %w", err)
		}
	}

	return nil
}

func (c *Circuit) Update(loadpoints []api.CircuitLoad) (err error) {
	defer func() {
		if c.maxPower != 0 && c.power > c.maxPower {
			c.log.WARN.Printf("over power detected: %.5gW > %.5gW", c.power, c.maxPower)
		} else {
			c.log.DEBUG.Printf("power: %.5gW", c.power)
		}

		if c.maxCurrent != 0 && c.current > c.maxCurrent {
			c.log.WARN.Printf("over current detected: %.3gA > %.3gA", c.current, c.maxCurrent)
		} else {
			c.log.DEBUG.Printf("current: %.3gA", c.current)
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
		potential := c.maxPower - c.power
		if delta > potential {
			capped := max(0, old+potential)
			c.log.DEBUG.Printf("validate power: %.5gW + (%.5gW -> %.5gW) > %.5gW capped at %.5gW", c.power, old, new, c.maxPower, capped)
			new = capped
		} else {
			c.log.TRACE.Printf("validate power: %.5gW + (%.5gW -> %.5gW) <= %.5gW ok", c.power, old, new, c.maxPower)
		}
	}

	if c.parent == nil {
		return new
	}

	return c.parent.ValidatePower(old, new)
}

// ValidateCurrent validates current request
func (c *Circuit) ValidateCurrent(old, new float64) float64 {
	delta := max(0, new-old)

	if c.maxCurrent != 0 {
		potential := c.maxCurrent - c.current
		if delta > potential {
			capped := max(0, old+potential)
			c.log.DEBUG.Printf("validate current: %.3gA + (%.3gA -> %.3gA) > %.3gA capped at %.3gA", c.current, old, new, c.maxCurrent, capped)
			new = capped
		} else {
			c.log.TRACE.Printf("validate current: %.3gA + (%.3gA -> %.3gA) <= %.3gA ok", c.current, old, new, c.maxCurrent)
		}
	}

	if c.parent == nil {
		return new
	}

	return c.parent.ValidateCurrent(old, new)
}
