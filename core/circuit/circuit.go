package circuit

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
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

	maxCurrent    float64                 // max allowed current
	maxPower      float64                 // max allowed power
	getMaxCurrent func() (float64, error) // dynamic max allowed current
	getMaxPower   func() (float64, error) // dynamic max allowed power

	current float64
	power   float64

	currentUpdated time.Time
	powerUpdated   time.Time
}

// NewFromConfig creates a new Circuit
func NewFromConfig(log *util.Logger, other map[string]interface{}) (api.Circuit, error) {
	cc := struct {
		Title         string           // title
		ParentRef     string           `mapstructure:"parent"` // parent circuit reference
		MeterRef      string           `mapstructure:"meter"`  // meter reference
		MaxCurrent    float64          // the max allowed current
		MaxPower      float64          // the max allowed power
		GetMaxCurrent *provider.Config // dynamic max allowed current
		GetMaxPower   *provider.Config // dynamic max allowed power
		Timeout       time.Duration    // timeout between meter updates
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

	circuit, err := New(log, cc.Title, cc.MaxCurrent, cc.MaxPower, meter, cc.Timeout)
	if err != nil {
		return nil, err
	}

	if cc.GetMaxPower != nil {
		res, err := provider.NewFloatGetterFromConfig(context.TODO(), *cc.GetMaxPower)
		if err != nil {
			return nil, err
		}
		circuit.getMaxPower = res
	}

	if cc.GetMaxCurrent != nil {
		res, err := provider.NewFloatGetterFromConfig(context.TODO(), *cc.GetMaxCurrent)
		if err != nil {
			return nil, err
		}
		circuit.getMaxCurrent = res
	}

	if cc.ParentRef != "" {
		dev, err := config.Circuits().ByName(cc.ParentRef)
		if err != nil {
			return nil, err
		}
		circuit.setParent(dev.Instance())
	}

	return circuit, err
}

// New creates a circuit
func New(log *util.Logger, title string, maxCurrent, maxPower float64, meter api.Meter, timeout time.Duration) (*Circuit, error) {
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

// setParent set parent circuit
func (c *Circuit) setParent(parent api.Circuit) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.parent != nil {
		return fmt.Errorf("circuit already has a parent")
	}
	c.parent = parent
	if parent != nil {
		parent.RegisterChild(c)
	}
	return nil
}

// Wrap wraps circuit with parent, keeping the original meter
func (c *Circuit) Wrap(parent api.Circuit) error {
	if c.meter != nil {
		parent.(*Circuit).meter = c.meter
	}
	return c.setParent(parent)
}

// HasMeter returns the max power setting
func (c *Circuit) HasMeter() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.meter != nil
}

// GetMaxPower returns the max power setting
func (c *Circuit) GetMaxPower() float64 {
	if c.getMaxPower != nil {
		res, err := c.getMaxPower()
		if err == nil {
			return res
		}
		c.log.WARN.Printf("get max power: %v", err)
	}

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
	if c.getMaxCurrent != nil {
		res, err := c.getMaxCurrent()
		if err == nil {
			return res
		}
		c.log.WARN.Printf("get max current: %v", err)
	}

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
		var p1, p2, p3 float64
		if phaseMeter, ok := c.meter.(api.PhasePowers); ok {
			var err error // phases needed for signed currents
			if p1, p2, p3, err = phaseMeter.Powers(); err != nil {
				return fmt.Errorf("circuit powers: %w", err)
			}
		}

		if i1, i2, i3, err := phaseMeter.Currents(); err == nil {
			c.current = max(util.SignFromPower(i1, p1), util.SignFromPower(i2, p2), util.SignFromPower(i3, p3))
			c.currentUpdated = time.Now()
		} else {
			c.overloadOnError(c.currentUpdated, &c.current)
			return fmt.Errorf("circuit currents: %w", err)
		}
	}

	return nil
}

func (c *Circuit) Update(loadpoints []api.CircuitLoad) (err error) {
	maxPower := c.GetMaxPower()
	maxCurrent := c.GetMaxCurrent()

	defer func() {
		if maxPower != 0 && c.power > maxPower {
			c.log.WARN.Printf("over power detected: %.5gW > %.5gW", c.power, maxPower)
		} else {
			c.log.DEBUG.Printf("power: %.5gW", c.power)
		}

		if maxCurrent != 0 && c.current > maxCurrent {
			c.log.WARN.Printf("over current detected: %.3gA > %.3gA", c.current, maxCurrent)
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

	if maxPower := c.GetMaxPower(); maxPower != 0 {
		potential := maxPower - c.power
		if delta > potential {
			capped := max(0, old+potential)
			c.log.DEBUG.Printf("validate power: %.5gW + (%.5gW -> %.5gW) > %.5gW capped at %.5gW", c.power, old, new, maxPower, capped)
			new = capped
		} else {
			c.log.TRACE.Printf("validate power: %.5gW + (%.5gW -> %.5gW) <= %.5gW ok", c.power, old, new, maxPower)
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

	if maxCurrent := c.GetMaxCurrent(); maxCurrent != 0 {
		potential := maxCurrent - c.current
		if delta > potential {
			capped := max(0, old+potential)
			c.log.DEBUG.Printf("validate current: %.3gA + (%.3gA -> %.3gA) > %.3gA capped at %.3gA", c.current, old, new, maxCurrent, capped)
			new = capped
		} else {
			c.log.TRACE.Printf("validate current: %.3gA + (%.3gA -> %.3gA) <= %.3gA ok", c.current, old, new, maxCurrent)
		}
	}

	if c.parent == nil {
		return new
	}

	return c.parent.ValidateCurrent(old, new)
}
