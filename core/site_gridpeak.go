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

const (
	gridPeakInterval    = 15 * time.Minute
	gridPeakMonthFormat = "2006-01"
)

// gridPeak tracks the month's highest clock-aligned 15-min average grid import
type gridPeak struct {
	Month string  `json:"month"` // calendar month of the peak
	Power float64 `json:"power"` // peak in W

	ts       time.Time // last sample
	power    float64   // last sample grid import in W
	energy   float64   // grid import of the current interval in Wh
	complete bool      // current interval observed from its start
}

// add integrates the last sample up to ts and returns true if a completed interval raised the month's peak
func (p *gridPeak) add(ts time.Time, power float64) bool {
	var raised bool

	if ts.Sub(p.ts) > gridPeakInterval {
		// gap, current interval cannot be trusted
		p.energy, p.complete = 0, false
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
	if month := start.Format(gridPeakMonthFormat); month != p.Month {
		p.Month, p.Power = month, 0
	}

	if power <= p.Power {
		return false
	}

	p.Power = power
	return true
}

// peak returns the peak of the month of ts
func (p *gridPeak) peak(ts time.Time) float64 {
	if ts.Format(gridPeakMonthFormat) != p.Month {
		return 0
	}
	return p.Power
}

// updateGridPeak tracks the month's grid import peak and the resulting target
func (site *Site) updateGridPeak(now time.Time) {
	if slices.Contains(gridPeakCountries, site.GetCountry()) {
		site.Lock()
		raised := site.gridPeak.add(now, site.gridPower)
		peak := site.gridPeak
		site.Unlock()

		if raised {
			site.log.DEBUG.Printf("grid peak: %.0fW", peak.Power)
			if err := settings.SetJson(keys.GridPeak, peak); err != nil {
				site.log.ERROR.Printf("grid peak: %v", err)
			}
		}

		site.publish(keys.GridPeak, peak.peak(now))
	}

	target := site.gridPeakLimit(now)

	site.Lock()
	site.gridPeakTarget = target
	site.Unlock()
}

// gridPeakLimiter supplies the grid import target to the root circuit
type gridPeakLimiter struct {
	site *Site
}

var _ api.ConsumptionLimiter = gridPeakLimiter{}

func (l gridPeakLimiter) MaxConsumptionPower() *float64 {
	l.site.RLock()
	defer l.site.RUnlock()
	res := l.site.gridPeakTarget
	return &res
}

// gridPeakLimit returns the grid import target, the month's peak floored at the baseline, 0 = none
func (site *Site) gridPeakLimit(now time.Time) float64 {
	baseline := site.GetGridPeakBaseline()
	if baseline <= 0 || !slices.Contains(gridPeakCountries, site.GetCountry()) {
		return 0
	}

	// fast charging accepts a higher peak
	if slices.ContainsFunc(site.loadpoints, func(lp *Loadpoint) bool {
		return lp.GetMode() == api.ModeNow
	}) {
		return 0
	}

	site.RLock()
	defer site.RUnlock()

	return max(baseline, site.gridPeak.peak(now))
}
