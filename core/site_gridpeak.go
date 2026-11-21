package core

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/tariff"
)

// shortcut: fixed time the loadpoints need to follow the slot budget, derive from interval and
// loadpoint count if slots still end above target
const gridPeakLag = time.Minute

// updateGridPeak derives the month's grid import peak from the grid meter's
// energy metrics and the resulting import target
func (site *Site) updateGridPeak() {
	var energy []*float64
	if c, ok := site.collectors[site.Meters.GridMeterRef]; ok {
		var err error
		if energy, err = c.DailyPeakEnergy(); err != nil {
			// keep the last target, an unknown peak must not lower it
			site.log.ERROR.Printf("grid peak: %v", err)
			return
		}
	}

	var peak float64
	days := make([]*float64, len(energy))
	for i, v := range energy {
		if v != nil {
			power := *v * 1e3 / tariff.SlotDuration.Hours()
			days[i] = &power
			peak = max(peak, power)
		}
	}

	// fast charging accepts a higher peak
	fast := slices.ContainsFunc(site.loadpoints, func(lp *Loadpoint) bool {
		return lp != nil && lp.GetMode() == api.ModeNow
	})

	site.Lock()
	var target float64
	if baseline := site.gridPeakBaseline; baseline > 0 && !fast {
		target = max(baseline, peak)
	}
	limit := target

	// a slot that imported above target so far must stay below it for the rest to keep its average
	if c, ok := site.collectors[site.Meters.GridMeterRef]; ok && target > 0 {
		if energy, elapsed, ok := c.SlotEnergy(); ok {
			// floor the time left, the loadpoints need it to follow
			remaining := max(tariff.SlotDuration-elapsed, gridPeakLag)
			budget := (target*tariff.SlotDuration.Hours() - energy*1e3) / remaining.Hours()
			// 0 means unlimited, an exhausted budget must not lift the limit
			limit = min(target, max(budget, 1))
		}
	}

	site.gridPeakTarget = target
	site.gridPeakLimit = limit
	site.Unlock()

	if limit < target {
		site.log.DEBUG.Printf("grid peak: slot budget limits import to %.0fW, target %.0fW", limit, target)
	}

	site.publish(keys.GridPeak, peak)
	site.publish(keys.GridPeakDays, days)
}

// getGridPeakLimit returns the grid import target capped by the running slot's budget in W (0 = none)
func (site *Site) getGridPeakLimit() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.gridPeakLimit
}

// gridPeakLimiter supplies a grid import limit to the root circuit
type gridPeakLimiter func() float64

func (f gridPeakLimiter) MaxConsumptionPower() *float64 {
	return new(f())
}
