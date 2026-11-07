package circuit

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/modbus"
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

	currents [3]float64 // per grid phase
	power    float64

	mappingMismatch int // consecutive updates where the meter contradicts the loadpoint phase mapping

	hems api.HEMS // only set on the root circuit, supplies the HEMS consumption cap

	currentUpdated time.Time
	powerUpdated   time.Time
}

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a new circuit from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Circuit, error) {
	cc := struct {
		Title         string         // title
		ParentRef     string         `mapstructure:"parent"` // parent circuit reference
		MeterRef      string         `mapstructure:"meter"`  // meter reference
		MaxCurrent    float64        // the max allowed current
		MaxPower      float64        // the max allowed power
		GetMaxCurrent *plugin.Config // dynamic max allowed current
		GetMaxPower   *plugin.Config // dynamic max allowed power
		Timeout       time.Duration  // timeout between meter updates
	}{
		Timeout: time.Minute,
	}

	// drop circuit type- all circuits are custom
	delete(other, "type")

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	var meter api.Meter
	if cc.MeterRef != "" {
		dev, err := config.Meters().ByName(cc.MeterRef)
		if err != nil {
			return nil, err
		}
		// disabled meter has no instance, circuit falls back to loadpoint sum
		meter = dev.Instance()
	}

	log := util.ContextLoggerWithDefault(ctx, util.NewLogger("circuit"))
	circuit, err := New(log, cc.Title, cc.MaxCurrent, cc.MaxPower, meter, cc.Timeout)
	if err != nil {
		return nil, err
	}

	circuit.getMaxPower, err = cc.GetMaxPower.FloatGetter(ctx)
	if err != nil {
		return nil, err
	}

	circuit.getMaxCurrent, err = cc.GetMaxCurrent.FloatGetter(ctx)
	if err != nil {
		return nil, err
	}

	if cc.ParentRef != "" {
		dev, err := config.Circuits().ByName(cc.ParentRef)
		if err != nil {
			return nil, err
		}
		parent := dev.Instance()
		if parent == nil {
			return nil, fmt.Errorf("missing parent circuit instance: %s", cc.ParentRef)
		}
		circuit.setParent(parent)
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
	} else if meter != nil && !api.HasCap[api.PhaseCurrents](meter) {
		return nil, errors.New("meter does not support phase currents")
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
	// prevent cyclical dependency
	for p := parent.GetParent(); p != nil; p = p.GetParent() {
		if c == p {
			return fmt.Errorf("cycle detected: %s and %s cannot be mutual parents", c.GetTitle(), parent.GetTitle())
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.parent != nil {
		return errors.New("circuit already has a parent")
	}
	c.parent = parent
	if parent != nil {
		parent.RegisterChild(c)
	}
	return nil
}

// SetHEMS attaches the HEMS instance to the circuit. Valid on root circuit only.
func (c *Circuit) SetHEMS(hems api.HEMS) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hems = hems
}

// HasMeter returns the max power setting
func (c *Circuit) HasMeter() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.meter != nil
}

// GetMaxPower returns the configured max power setting.
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

// EffectiveMaxPower returns the configured max power clamped by the HEMS
// consumption limit. Only the root circuit applies the HEMS clamp; non-root
// callers walk up the parent chain via ValidatePower.
func (c *Circuit) EffectiveMaxPower() float64 {
	maxPower := c.GetMaxPower()
	if c.hems == nil {
		return maxPower
	}

	hemsLimit := c.hems.MaxConsumptionPower()
	if hemsLimit == nil || *hemsLimit <= 0 {
		return maxPower
	}

	// unconfigured maxPower means unlimited, so the HEMS limit applies alone
	if maxPower <= 0 {
		return *hemsLimit
	}

	return min(*hemsLimit, maxPower)
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
	c.currents = [3]float64{}

	for _, lp := range loadpoints {
		if lp.GetCircuit() != c {
			continue
		}

		c.power += lp.GetChargePower()
		c.addCurrents(lp.GetGridPhaseCurrents())
	}
}

func (c *Circuit) addCurrents(currents [3]float64) {
	for i := range c.currents {
		c.currents[i] += currents[i]
	}
}

func (c *Circuit) overloadOnError(t time.Time, val *float64) {
	if c.timeout > 0 && time.Since(t) > c.timeout {
		*val = math.MaxFloat64
	}
}

func (c *Circuit) overloadCurrentsOnError(t time.Time) {
	if c.timeout > 0 && time.Since(t) > c.timeout {
		c.currents = [3]float64{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}
	}
}

// mismatchTolerance is the measurement and timing deviation accepted before the
// meter is considered to contradict the configured loadpoint phase mapping
func mismatchTolerance(expected float64) float64 {
	return max(2, 0.2*expected)
}

// mismatchCycles is the number of consecutive updates a contradiction must persist
const mismatchCycles = 3

// checkPhaseMapping warns if the meter measures less current on a grid phase than
// the loadpoints with a configured phase mapping draw from it. Other loads on the
// circuit only add current, so a lower reading means a wrong mapping.
// Root circuits are skipped since a grid meter also sees local generation.
func (c *Circuit) checkPhaseMapping(loadpoints []api.CircuitLoad) {
	// a meter for a power limit only may not measure phase currents
	if c.parent == nil || !api.HasCap[api.PhaseCurrents](c.meter) {
		return
	}

	var expected [3]float64
	var mapped bool

	// loadpoints without a reliable measurement are left out, the sum of the
	// others still is a lower bound of what the meter must see
	for _, lp := range loadpoints {
		if lp.GetCircuit() != c {
			continue
		}

		currents, ok := lp.MeasuredGridPhaseCurrents()
		if !ok {
			continue
		}

		mapped = true
		for i, v := range currents {
			expected[i] += v
		}
	}

	if !mapped {
		return
	}

	for i := range expected {
		if c.currents[i]+mismatchTolerance(expected[i]) < expected[i] {
			c.mappingMismatch++
			if c.mappingMismatch == mismatchCycles {
				c.log.WARN.Printf("loadpoint phase mapping does not match meter: L%d measured %.3gA, expected at least %.3gA", i+1, c.currents[i], expected[i])
			}
			return
		}
	}

	c.mappingMismatch = 0
}

func (c *Circuit) updateMeters() error {
	if f, err := backoff.RetryWithData(c.meter.CurrentPower, modbus.Backoff()); err == nil {
		c.power = f
		c.powerUpdated = time.Now()
	} else {
		c.overloadOnError(c.powerUpdated, &c.power)
		return fmt.Errorf("circuit power: %w", err)
	}

	if phaseMeter, ok := api.Cap[api.PhaseCurrents](c.meter); ok {
		var i1, i2, i3 float64
		if err := backoff.Retry(func() error {
			var err error
			i1, i2, i3, err = phaseMeter.Currents()
			return err
		}, modbus.Backoff()); err != nil {
			c.overloadCurrentsOnError(c.currentUpdated)
			return fmt.Errorf("circuit currents: %w", err)
		}

		var p1, p2, p3 float64
		if phaseMeter, ok := api.Cap[api.PhasePowers](c.meter); ok {
			var err error // phases needed for signed currents
			if p1, p2, p3, err = phaseMeter.Powers(); err != nil {
				return fmt.Errorf("circuit powers: %w", err)
			}
		}

		c.currents = [3]float64{util.SignFromPower(i1, p1), util.SignFromPower(i2, p2), util.SignFromPower(i3, p3)}
		c.currentUpdated = time.Now()
	}

	return nil
}

func (c *Circuit) Update(loadpoints []api.CircuitLoad) (err error) {
	maxPower := c.GetMaxPower()
	maxCurrent := c.GetMaxCurrent()

	defer func() {
		if maxPower != 0 && c.power > maxPower {
			c.log.WARN.Printf("over power detected: %.0fW > %.0fW", c.power, maxPower)
		} else {
			c.log.DEBUG.Printf("power: %.0fW", c.power)
		}

		if current := c.maxPhaseCurrent(); maxCurrent != 0 && current > maxCurrent {
			c.log.WARN.Printf("over current detected: %.3gA > %.3gA", current, maxCurrent)
		} else {
			c.log.DEBUG.Printf("current: %.3gA", current)
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
		if err := c.updateMeters(); err != nil {
			return err
		}
		c.checkPhaseMapping(loadpoints)
		return nil
	}

	// no meter available
	c.updateLoadpoints(loadpoints)
	for _, ch := range c.children {
		c.power += ch.GetChargePower()
		c.addCurrents(ch.GetGridPhaseCurrents())
	}

	return nil
}

// GetChargePower returns the actual power
func (c *Circuit) GetChargePower() float64 {
	return c.power
}

func (c *Circuit) maxPhaseCurrent() float64 {
	return max(c.currents[0], c.currents[1], c.currents[2])
}

// GetMaxPhaseCurrent returns the actual current of the highest loaded phase
func (c *Circuit) GetMaxPhaseCurrent() float64 {
	return c.maxPhaseCurrent()
}

// GetGridPhaseCurrents returns the actual currents per grid phase
func (c *Circuit) GetGridPhaseCurrents() [3]float64 {
	return c.currents
}

// ValidatePower validates power request
func (c *Circuit) ValidatePower(old, new float64) float64 {
	if maxPower := c.EffectiveMaxPower(); maxPower != 0 {
		delta := max(0, new-old)
		potential := maxPower - c.power

		if delta > potential {
			capped := min(new, max(0, old+potential))
			c.log.DEBUG.Printf("validate power: %.0fW + (%.0fW -> %.0fW) > %.0fW capped at %.0fW", c.power, old, new, maxPower, capped)
			new = capped
		} else {
			c.log.TRACE.Printf("validate power: %.0fW + (%.0fW -> %.0fW) <= %.0fW ok", c.power, old, new, maxPower)
		}
	}

	if c.parent == nil {
		return new
	}

	return c.parent.ValidatePower(old, new)
}

// ValidateCurrent validates a current request on the given grid phases. old is the
// current the load draws per grid phase now: every phase is checked on its own and
// only the headroom left by the other loads on that phase counts.
func (c *Circuit) ValidateCurrent(phases api.GridPhases, old [3]float64, new float64) float64 {
	if maxCurrent := c.GetMaxCurrent(); maxCurrent != 0 {
		for i, active := range phases {
			if !active {
				continue
			}

			if limit := max(0, old[i]+maxCurrent-c.currents[i]); new > limit {
				c.log.DEBUG.Printf("validate current: L%d %.3gA + (%.3gA -> %.3gA) > %.3gA capped at %.3gA", i+1, c.currents[i], old[i], new, maxCurrent, limit)
				new = limit
			}
		}
	}

	if c.parent == nil {
		return new
	}

	return c.parent.ValidateCurrent(phases, old, new)
}
