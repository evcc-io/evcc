package prioritizer

import (
	"fmt"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

// Settings provides the site-level priority strategy configuration
type Settings interface {
	// GetPriorityStrategy returns the loadpoint priority sub-ordering strategy
	GetPriorityStrategy() api.PriorityStrategy
	// GetPriorityHysteresis returns the priority sub-ordering deadband (soc-% or kWh per basis)
	GetPriorityHysteresis() int
	// EffectivePriorityScoring returns the site-wide basis and the reference value the strategy gap is normalised against
	EffectivePriorityScoring() (api.PriorityBasis, float64)
}

type Prioritizer struct {
	mu       sync.Mutex
	log      *util.Logger
	settings Settings
	demand   map[loadpoint.API]float64
}

func New(log *util.Logger, settings Settings) *Prioritizer {
	return &Prioritizer{
		log:      log,
		settings: settings,
		demand:   make(map[loadpoint.API]float64),
	}
}

func (p *Prioritizer) UpdateChargePowerFlexibility(lp loadpoint.API, rates api.Rates) {
	if power := lp.GetChargePowerFlexibility(rates); power >= 0 {
		p.mu.Lock()
		p.demand[lp] = power
		p.mu.Unlock()
	}
}

func (p *Prioritizer) GetChargePowerFlexibility(lp loadpoint.API) float64 {
	// strategy, basis and reference are site-level, so every loadpoint is scored on one scale
	strategy := p.settings.GetPriorityStrategy()
	basis, ref := p.settings.EffectivePriorityScoring()
	score := lp.EffectivePriorityScore(strategy, basis, ref)
	prio, heating := lp.EffectivePriority(), lp.IsHeating()

	// hysteresis deadband in gap units (soc-% or kWh), normalised against the same reference
	// as the score fraction so near-equal loadpoints tie and converge instead of leapfrogging
	band := float64(p.settings.GetPriorityHysteresis()) / ref

	var (
		reduceBy float64
		msg      strings.Builder
	)

	for other, power := range p.demand {
		if power <= 0 {
			continue
		}

		// the deadband sub-orders within a tier only - an explicit priority always wins.
		// heating aliases temperature as soc and carries no comparable score, so a
		// same-tier pair involving heating is left untouched instead of always losing.
		var threshold float64
		if prio == other.EffectivePriority() {
			if heating || other.IsHeating() {
				continue
			}
			threshold = band
		}

		otherScore := other.EffectivePriorityScore(strategy, basis, ref)
		if score-otherScore > threshold {
			reduceBy += power
			msg.WriteString(fmt.Sprintf("%.0fW from %s at prio %.2f, ", power, other.GetTitle(), otherScore))
		}
	}

	if p.log != nil && reduceBy > 0 {
		p.log.DEBUG.Printf("lp %s at prio %.2f gets additional %stotal %.0fW\n", lp.GetTitle(), score, msg.String(), reduceBy)
	}

	return reduceBy
}
