package core

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
)

// gridPeakCountries bill the month's highest clock-aligned 15-min average grid import
var gridPeakCountries = []string{"AT", "BE"}

const gridPeakInterval = 15 * time.Minute

// gridPeak tracks the month's highest clock-aligned 15-min average grid import
type gridPeak struct {
	Time  time.Time `json:"time"`  // start of the peak interval
	Power float64   `json:"power"` // peak in W

	ts       time.Time // last sample
	power    float64   // last sample grid import in W
	energy   float64   // grid import of the current interval in Wh
	complete bool      // current interval observed from its start
}

// add integrates the last sample up to ts and returns true if a completed interval raised the month's peak
func (p *gridPeak) add(ts time.Time, power float64) bool {
	var raised bool

	if ts.Sub(p.ts) > gridPeakInterval {
		// gap, current interval cannot be trusted unless it starts now
		p.energy, p.complete = 0, ts.Equal(ts.Truncate(gridPeakInterval))
	} else {
		for t := p.ts; t.Before(ts); {
			end := t.Truncate(gridPeakInterval).Add(gridPeakInterval)
			if end.After(ts) {
				end = ts
			}

			p.energy += p.power * end.Sub(t).Hours()

			if end.Equal(end.Truncate(gridPeakInterval)) {
				if p.complete && p.raise(end.Add(-gridPeakInterval), p.energy/gridPeakInterval.Hours()) {
					raised = true
				}
				p.energy, p.complete = 0, true
			}

			t = end
		}
	}

	p.ts, p.power = ts, max(0, power)

	return raised
}

// raise records the average of the interval starting at start if it exceeds its month's peak
func (p *gridPeak) raise(start time.Time, power float64) bool {
	if power <= p.peak(start) {
		return false
	}

	p.Time, p.Power = start, power
	return true
}

// peak returns the peak of the month of ts
func (p *gridPeak) peak(ts time.Time) float64 {
	if !sameMonth(ts, p.Time) {
		return 0
	}
	return p.Power
}

// sameMonth reports whether a and b fall in the same calendar month in the location of a
func sameMonth(a, b time.Time) bool {
	ay, am, _ := a.Date()
	by, bm, _ := b.In(a.Location()).Date()
	return ay == by && am == bm
}

// updateGridPeak tracks the month's grid import peak and the resulting import target
func (site *Site) updateGridPeak(now time.Time) {
	regime := slices.Contains(gridPeakCountries, site.GetCountry())
	baseline := site.GetGridPeakBaseline()

	// fast charging accepts a higher peak
	fast := slices.ContainsFunc(site.loadpoints, func(lp *Loadpoint) bool {
		return lp.GetMode() == api.ModeNow
	})

	site.Lock()
	var raised bool
	if regime {
		if site.gridPowerUpdated.After(site.gridPeak.ts) {
			raised = site.gridPeak.add(now, site.gridPower)
		} else {
			// no fresh grid meter reading, current interval cannot be trusted
			site.gridPeak.ts = time.Time{}
		}
	}
	peak := site.gridPeak
	site.gridPeakTarget = 0
	if regime && baseline > 0 && !fast {
		site.gridPeakTarget = max(baseline, peak.peak(now))
	}
	site.Unlock()

	if !regime {
		return
	}

	if raised {
		site.log.DEBUG.Printf("grid peak: %.0fW", peak.Power)
		if err := settings.SetJson(keys.GridPeak, peak); err != nil {
			site.log.ERROR.Printf("grid peak: %v", err)
		}
	}

	site.publish(keys.GridPeak, peak.peak(now))
}

// getGridPeakTarget returns the grid import target in W, 0 = none
func (site *Site) getGridPeakTarget() float64 {
	site.RLock()
	defer site.RUnlock()
	return site.gridPeakTarget
}

// gridPeakLimiter supplies the grid import target to the root circuit
type gridPeakLimiter struct {
	site *Site
}

func (l gridPeakLimiter) MaxConsumptionPower() *float64 {
	return new(l.site.getGridPeakTarget())
}
