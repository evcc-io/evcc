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

// updateCircuitBattery passes the battery grid charge power to the root circuit.
// Only batteries that evcc put into charge mode count, the circuit decides if the
// power is flexible for managed loadpoints.
func (site *Site) updateCircuitBattery() {
	if site.circuit == nil {
		return
	}

	var power float64
	var charging []api.Meter

	site.RLock()
	for i, dev := range site.batteryMeters {
		if i < len(site.battery.Devices) && site.batteryModeApplied[dev.Config().Name] == api.BatteryCharge {
			// negative battery power is charging
			if p := -site.battery.Devices[i].Power; p > 0 {
				power += p
				charging = append(charging, dev.Instance())
			}
		}
	}
	site.RUnlock()

	// the phases of the battery are unknown without measured currents, assuming
	// a symmetric split would free current on phases the battery does not load
	var currents [3]float64
	for _, meter := range charging {
		pc, ok := api.Cap[api.PhaseCurrents](meter)
		if !ok {
			currents = [3]float64{}
			break
		}

		i1, i2, i3, err := pc.Currents()
		if err != nil {
			site.log.DEBUG.Printf("battery currents: %v", err)
			currents = [3]float64{}
			break
		}

		currents[0] += math.Abs(i1)
		currents[1] += math.Abs(i2)
		currents[2] += math.Abs(i3)
	}

	site.circuit.SetBatteryCharge(power, currents[0], currents[1], currents[2])
}

// applyHemsLimits applies the HEMS dim and curtail state to the site's devices
func (site *Site) applyHemsLimits() {
	if site.hems == nil {
		return
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		if dim := hems.Dimmed(site.hems); dim != nil {
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

// dimMeters applies the HEMS dim state to all dimmable aux and ext meters.
// Devices are only queried when the state changes or after a failed attempt.
func (site *Site) dimMeters(dim bool) error {
	if site.dimmed != nil && *site.dimmed == dim {
		return nil
	}

	// invalidate until successfully applied
	site.dimmed = nil

	var errs error
	for _, dev := range slices.Concat(site.auxMeters, site.extMeters) {
		m, ok := api.Cap[api.Dimmer](dev.Instance())
		if !ok {
			continue
		}

		// unreadable state: apply unconditionally
		dimmed, err := backoff.RetryWithData(m.Dimmed, modbus.Backoff())
		if err != nil && !errors.Is(err, api.ErrNotAvailable) {
			errs = errors.Join(errs, fmt.Errorf("%s dimmed: %w", deviceTitleOrName(dev), err))
			continue
		}
		if err == nil && dim == dimmed {
			continue
		}

		if err := m.Dim(dim); err == nil {
			site.log.DEBUG.Printf("%s dim: %t", deviceTitleOrName(dev), dim)
		} else if !errors.Is(err, api.ErrNotAvailable) {
			errs = errors.Join(errs, fmt.Errorf("%s dim: %w", deviceTitleOrName(dev), err))
		}
	}

	if errs == nil {
		site.dimmed = &dim
	}

	return errs
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
		res = append(res, curtailable{deviceTitleOrName(dev), dev.Instance()})
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
