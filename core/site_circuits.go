package core

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/samber/lo"
)

type circuitStruct struct {
	Name       string   `json:"name,omitempty"`
	Title      string   `json:"title,omitempty"`
	Icon       string   `json:"icon,omitempty"`
	Parent     string   `json:"parent,omitempty"`
	Power      float64  `json:"power"`
	Current    *float64 `json:"current,omitempty"`
	MaxPower   float64  `json:"maxPower,omitempty"`
	MaxCurrent float64  `json:"maxCurrent,omitempty"`
}

// updateCircuits updates all circuits' power and currents
func (site *Site) updateCircuits() {
	if site.circuit == nil {
		return
	}

	if err := site.circuit.Update(site.loadpointsAsCircuitDevices()); err != nil {
		site.log.ERROR.Println(err)
	}

	site.publishCircuits()
}

// applyHemsLimits applies the HEMS dim and curtail state to the site's devices
func (site *Site) applyHemsLimits() {
	if site.hems == nil {
		return
	}

	site.dimState = site.hemsDimState()

	var wg sync.WaitGroup

	wg.Go(func() {
		if dim := site.dimState; dim != nil {
			if err := site.dimMeters(*dim); err != nil {
				site.log.ERROR.Println(err)
			}
		}
	})

	wg.Go(func() {
		if hems.Curtailed(site.hems) != nil {
			if err := site.curtailPV(site.hems.CurtailedPercent()); err != nil {
				site.log.ERROR.Println(err)
			}
		}
	})

	wg.Wait()
}

// publishCircuits returns a list of circuit titles
func (site *Site) publishCircuits() {
	cc := config.Circuits().Devices()
	res := make(map[string]circuitStruct, len(cc))

	for _, c := range cc {
		instance := c.Instance()
		if instance == nil {
			continue
		}
		props := deviceProperties(c)

		// config reference instead of instance: an updated device keeps its name but gets a new instance
		parent, _ := c.Config().Property("parent").(string)

		data := circuitStruct{
			Name:       c.Config().Name,
			Title:      cmp.Or(props.Title, instance.GetTitle()),
			Icon:       props.Icon,
			Parent:     parent,
			Power:      instance.GetChargePower(),
			MaxPower:   instance.GetMaxPower(),
			MaxCurrent: instance.GetMaxCurrent(),
		}

		if instance.GetMaxCurrent() > 0 {
			data.Current = lo.EmptyableToPtr(instance.GetMaxPhaseCurrent())
		}

		res[c.Config().Name] = data
	}

	site.publish(keys.Circuits, res)
}

// dimState is the HEMS dim decision applied to all dimmers
type dimState struct {
	active   bool
	fallback bool // configured dim powers exceed the HEMS limit, all dimmers dim to 0W
}

// limit returns the residual consumption power the dimmer may keep while dimmed
func (d dimState) limit(dimmer api.Dimmer) float64 {
	if !d.active || d.fallback {
		return 0
	}
	return dimPower(dimmer)
}

// dimActive returns whether the dimmer is dimmed, exempt dimmers are never dimmed
func (d dimState) dimActive(dimmer api.Dimmer) bool {
	return d.active && !dimExempt(dimmer)
}

// dimExempt returns whether the dimmer is exempt from dimming by a finite negative dim power
func dimExempt(dimmer api.Dimmer) bool {
	p := dimmer.DimPower()
	return p < 0 && !math.IsInf(p, -1)
}

// dimPower returns the configured residual power, treating exempt and invalid values as 0W
func dimPower(dimmer api.Dimmer) float64 {
	p := dimmer.DimPower()
	if !(p > 0) || math.IsInf(p, 0) {
		return 0
	}
	return p
}

// dimmers returns all dimmable aux and ext meters and loadpoint chargers
func (site *Site) dimmers() []api.Dimmer {
	var res []api.Dimmer

	for _, dev := range slices.Concat(site.auxMeters, site.extMeters) {
		if m, ok := api.Cap[api.Dimmer](dev.Instance()); ok {
			res = append(res, m)
		}
	}

	for _, lp := range site.activeLoadpoints() {
		if m, ok := api.Cap[api.Dimmer](lp.charger); ok {
			res = append(res, m)
		}
	}

	return res
}

// hemsDimState returns the HEMS dim decision. If the configured dim powers exceed
// the HEMS limit, all dimmers fall back to 0W.
func (site *Site) hemsDimState() *dimState {
	dim := hems.Dimmed(site.hems)
	if dim == nil {
		return nil
	}

	res := &dimState{active: *dim}
	if !res.active {
		return res
	}

	limit := site.hems.MaxConsumptionPower()
	if limit == nil {
		return res
	}

	var total float64
	for _, d := range site.dimmers() {
		total += dimPower(d)
	}

	if total > *limit {
		res.fallback = true

		if site.dimState == nil || !site.dimState.fallback {
			site.log.ERROR.Printf("dim power %.0fW exceeds limit %.0fW, dimming all devices to 0W", total, *limit)
		}
	}

	return res
}

// dimMeters applies the HEMS dim state to all dimmable aux and ext meters.
// Devices are only queried when the state or the set of dimmers changes, or after a failed attempt.
func (site *Site) dimMeters(dim dimState) error {
	type namedDimmer struct {
		name string
		api.Dimmer
	}

	var dimmers []namedDimmer
	for _, dev := range slices.Concat(site.auxMeters, site.extMeters) {
		if m, ok := api.Cap[api.Dimmer](dev.Instance()); ok {
			dimmers = append(dimmers, namedDimmer{deviceTitleOrName(dev), m})
		}
	}

	// a dimmer appearing later, e.g. on EEBus reconnect, must receive the current state
	if site.dimmed != nil && site.dimmed.state == dim && site.dimmed.devices == len(dimmers) {
		return nil
	}

	// invalidate until successfully applied
	site.dimmed = nil

	var errs error
	for _, m := range dimmers {
		// unreadable state: apply unconditionally
		dimmed, err := backoff.RetryWithData(m.Dimmed, modbus.Backoff())
		if err != nil && !errors.Is(err, api.ErrNotAvailable) {
			errs = errors.Join(errs, fmt.Errorf("%s dimmed: %w", m.name, err))
			continue
		}
		// an active state is always written since the device may hold a different limit
		active := dim.dimActive(m)
		if err == nil && !active && !dimmed {
			continue
		}

		limit := dim.limit(m)
		if err := m.Dim(active, limit); err == nil {
			site.log.DEBUG.Printf("%s dim: %t (%.0fW)", m.name, active, limit)
		} else if !errors.Is(err, api.ErrNotAvailable) {
			errs = errors.Join(errs, fmt.Errorf("%s dim: %w", m.name, err))
		}
	}

	if errs == nil {
		site.dimmed = &dimApplied{state: dim, devices: len(dimmers)}
	}

	return errs
}

// dimApplied is the dim state last applied to the dimmable meters
type dimApplied struct {
	state   dimState
	devices int
}

// curtailable is a named curtailment device
type curtailable struct {
	name string
	api.Curtailer
}

// curtailables returns the curtailment devices and the curtailable pv meters
func (site *Site) curtailables() []curtailable {
	var res []curtailable

	for _, dev := range site.pvMeters {
		if m, ok := api.Cap[api.Curtailer](dev.Instance()); ok {
			res = append(res, curtailable{deviceTitleOrName(dev), m})
		}
	}

	for _, dev := range site.curtailers {
		if dev.Instance() != nil {
			res = append(res, curtailable{deviceTitleOrName(dev), dev.Instance()})
		}
	}

	return res
}

// curtailPV applies the HEMS curtailment percent to all curtailment devices and curtailable pv meters.
// Devices are only queried when the percent changes or after a failed attempt.
func (site *Site) curtailPV(percent *int) error {
	if percent == nil || site.curtailPercent != nil && *site.curtailPercent == *percent {
		return nil
	}

	// invalidate until successfully applied
	site.curtailPercent = nil

	var errs error
	for _, m := range site.curtailables() {
		// unreadable state: apply unconditionally
		curtailed, err := backoff.RetryWithData(m.CurtailedPercent, modbus.Backoff())
		if err != nil && !errors.Is(err, api.ErrNotAvailable) {
			errs = errors.Join(errs, fmt.Errorf("%s curtailed: %w", m.name, err))
			continue
		}
		if err == nil && curtailed == *percent {
			continue
		}

		if err := m.SetCurtailPercent(*percent); err == nil {
			site.log.DEBUG.Printf("%s curtail: %d%%", m.name, *percent)
		} else if !errors.Is(err, api.ErrNotAvailable) {
			errs = errors.Join(errs, fmt.Errorf("%s curtail: %w", m.name, err))
		}
	}

	if errs == nil {
		site.curtailPercent = new(*percent)
	}

	return errs
}
