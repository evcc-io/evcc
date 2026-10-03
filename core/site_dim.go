package core

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util/modbus"
)

// dimmable is a named §14a device with its share of the consumption budget
type dimmable struct {
	name string
	api.Dimmer
	lp                 loadpoint.API // set for loadpoint chargers
	priority           int
	minPower, maxPower float64
}

// dimmables returns the dimmable loadpoint chargers and aux/ext meters
func (site *Site) dimmables() []dimmable {
	var res []dimmable

	for _, lp := range site.loadpoints {
		if d, ok := api.Cap[api.Dimmer](lp.charger); ok {
			maxPower := lp.EffectiveMaxPower()
			if maxPower <= 0 {
				maxPower = math.Inf(1)
			}
			res = append(res, dimmable{lp.GetTitle(), d, lp, lp.EffectivePriority(), lp.EffectiveMinPower(), maxPower})
		}
	}

	for _, dev := range slices.Concat(site.auxMeters, site.extMeters) {
		if d, ok := api.Cap[api.Dimmer](dev.Instance()); ok {
			res = append(res, dimmable{deviceTitleOrName(dev), d, nil, 0, 0, math.Inf(1)})
		}
	}

	return res
}

// allocateDim splits the §14a budget across the devices: higher priority is served first,
// equal priority shares the remainder evenly up to each device's max power. A share below
// a device's min power switches it off (0) instead of handing it a limit it cannot use.
func allocateDim(budget float64, devs []dimmable) []float64 {
	res := make([]float64, len(devs))

	idx := make([]int, len(devs))
	for i := range idx {
		idx[i] = i
	}
	// priority descending, then smallest demand first so its unused share flows to the larger ones
	slices.SortStableFunc(idx, func(a, b int) int {
		if c := devs[b].priority - devs[a].priority; c != 0 {
			return c
		}
		return cmpFloat(devs[a].maxPower, devs[b].maxPower)
	})

	for i := 0; i < len(idx); {
		// equal-priority group
		j := i
		for j < len(idx) && devs[idx[j]].priority == devs[idx[i]].priority {
			j++
		}

		for n := j - i; i < j; i, n = i+1, n-1 {
			d := devs[idx[i]]
			share := min(d.maxPower, budget/float64(n))
			if share < d.minPower {
				share = 0
			}
			res[idx[i]] = share
			budget -= share
		}
	}

	return res
}

func cmpFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// dimDevices applies the HEMS consumption limit to all dimmable devices, like curtailPV
// for curtailment: 0 releases every device, otherwise the budget is allocated across them.
// Devices are only queried when their limit changes or after a failed attempt.
func (site *Site) dimDevices(limit float64) error {
	devs := site.dimmables()

	limits := make([]float64, len(devs))
	if limit > 0 {
		limits = allocateDim(limit, devs)
	} else {
		for i := range limits {
			limits[i] = math.Inf(1)
		}
	}

	// loadpoints pause control while their charger is limited
	site.dimmed = make(map[loadpoint.API]bool, len(devs))
	for i, d := range devs {
		if d.lp != nil {
			site.dimmed[d.lp] = !math.IsInf(limits[i], 1)
		}
	}

	if site.dimLimits != nil && slices.Equal(site.dimLimits, limits) {
		return nil
	}

	// invalidate until successfully applied
	site.dimLimits = nil

	var errs error
	for i, d := range devs {
		limit := limits[i]
		dim := !math.IsInf(limit, 1)

		// unreadable state: apply unconditionally
		dimmed, err := backoff.RetryWithData(d.Dimmed, modbus.Backoff())
		if err != nil && !errors.Is(err, api.ErrNotAvailable) {
			errs = errors.Join(errs, fmt.Errorf("%s dimmed: %w", d.name, err))
			continue
		}
		// released on both sides: nothing to write; an active limit is re-stated since its value may have changed
		if err == nil && !dim && !dimmed {
			continue
		}

		if err := d.Dim(limit); err == nil {
			site.log.DEBUG.Printf("%s dim: %t (%.0fW)", d.name, dim, limit)
		} else if !errors.Is(err, api.ErrNotAvailable) {
			errs = errors.Join(errs, fmt.Errorf("%s dim: %w", d.name, err))
		}
	}

	if errs == nil {
		site.dimLimits = limits
	}

	return errs
}

// loadpointDimmed returns the §14a state of the loadpoint's charger as applied by dimDevices,
// nil when the HEMS made no statement or the charger is not dimmable
func (site *Site) loadpointDimmed(lp loadpoint.API) *bool {
	if dimmed, ok := site.dimmed[lp]; ok {
		return &dimmed
	}
	return nil
}
