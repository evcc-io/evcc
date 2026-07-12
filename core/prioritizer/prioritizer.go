package prioritizer

import (
	"fmt"
	"math"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

// Settings provides the site-level priority strategy configuration
type Settings interface {
	// GetPriorityStrategy returns the loadpoint priority sub-ordering strategy
	GetPriorityStrategy() api.PriorityStrategy
	// GetPriorityBasis returns the priority strategy basis (percent, energy)
	GetPriorityBasis() api.PriorityBasis
	// GetPriorityHysteresis returns the priority sub-ordering deadband (soc-% or kWh per basis)
	GetPriorityHysteresis() int
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
	// strategy and basis are site-level settings, so every candidate is scored
	// on one scale (see effectiveBasis)
	strategy := p.settings.GetPriorityStrategy()
	basis := p.effectiveBasis(lp)
	score := lp.EffectivePriorityScore(strategy, basis)

	// hysteresis deadband (soc-% -> score fraction): only outrank another loadpoint
	// when ahead by more than the band, so near-equal soc loadpoints tie and converge
	// instead of leapfrogging each other as their soc crosses. capped below 1.0 so it
	// never weakens cross-tier (integer priority) ordering.
	band := math.Min(float64(p.settings.GetPriorityHysteresis())/100, 0.99)

	var (
		reduceBy float64
		msg      string
	)

	for other, power := range p.demand {
		otherScore := other.EffectivePriorityScore(strategy, basis)
		if score-otherScore > band && power > 0 {
			reduceBy += power
			msg += fmt.Sprintf("%.0fW from %s at prio %.2f, ", power, other.GetTitle(), otherScore)
		}
	}

	if p.log != nil && reduceBy > 0 {
		p.log.DEBUG.Printf("lp %s at prio %.2f gets additional %stotal %.0fW\n", lp.GetTitle(), score, msg, reduceBy)
	}

	return reduceBy
}

// candidates returns the loadpoints that participate in ranking: the target plus
// every loadpoint that has registered demand.
func (p *Prioritizer) candidates(lp loadpoint.API) []loadpoint.API {
	res := []loadpoint.API{lp}
	for other := range p.demand {
		if other != lp {
			res = append(res, other)
		}
	}
	return res
}

// effectiveBasis returns the priority basis to score all loadpoints with. The energy
// basis ranks by absolute kWh while the percent basis ranks by soc-%; their fractions
// are not comparable, so the whole comparison set must use a single basis. When any
// participating loadpoint has no known vehicle capacity (its energy score would
// silently fall back to a percentage), the entire set is ranked by percent so
// configured and unconfigured vehicles are never mixed across scales.
func (p *Prioritizer) effectiveBasis(lp loadpoint.API) api.PriorityBasis {
	basis := p.settings.GetPriorityBasis()
	if basis != api.PriorityBasisEnergy {
		return basis
	}

	for _, other := range p.candidates(lp) {
		if v := other.GetVehicle(); v == nil || v.Capacity() <= 0 {
			return api.PriorityBasisPercent
		}
	}

	return api.PriorityBasisEnergy
}
