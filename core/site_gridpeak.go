package core

import (
	"slices"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/tariff"
)

// updateGridPeak derives the month's grid import peak from the grid meter's
// energy metrics and the resulting import target
func (site *Site) updateGridPeak() {
	var energy []*float64
	if c, ok := site.collectors[site.Meters.GridMeterRef]; ok {
		var err error
		if energy, err = c.DailyPeakEnergy(); err != nil {
			site.log.ERROR.Printf("grid peak: %v", err)
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

	baseline := site.GetGridPeakBaseline()

	site.Lock()
	site.gridPeakTarget = 0
	if baseline > 0 && !fast {
		site.gridPeakTarget = max(baseline, peak)
	}
	site.Unlock()

	site.publish(keys.GridPeak, peak)
	site.publish(keys.GridPeakDays, days)
}

// gridPeakLimiter supplies the grid import target to the root circuit
type gridPeakLimiter struct {
	site *Site
}

func (l gridPeakLimiter) MaxConsumptionPower() *float64 {
	return new(l.site.GetGridPeakTarget())
}
