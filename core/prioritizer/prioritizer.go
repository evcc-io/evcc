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
	leads    map[[2]loadpoint.API]bool
}

func New(log *util.Logger, settings Settings) *Prioritizer {
	return &Prioritizer{
		log:      log,
		settings: settings,
		demand:   make(map[loadpoint.API]float64),
		leads:    make(map[[2]loadpoint.API]bool),
	}
}

func (p *Prioritizer) UpdateChargePowerFlexibility(lp loadpoint.API, rates api.Rates) {
	if power := lp.GetChargePowerFlexibility(rates); power >= 0 {
		p.mu.Lock()
		p.demand[lp] = power
		p.mu.Unlock()
	}
}

// Outranks reports whether a takes surplus before b by tier, then by a latched strategy gap.
// Unavailable gaps tie; the current winner holds until the challenger exceeds hysteresis.
func (p *Prioritizer) Outranks(a, b loadpoint.API) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.outranks(a, b)
}

func (p *Prioritizer) outranks(a, b loadpoint.API) bool {
	if pa, pb := a.EffectivePriority(), b.EffectivePriority(); pa != pb {
		return pa > pb
	}

	strategy, basis := p.settings.GetPriorityStrategy(), p.settings.GetPriorityBasis()
	ga, oka := a.PriorityGap(strategy, basis)
	gb, okb := b.PriorityGap(strategy, basis)
	if !oka || !okb {
		return false
	}

	key, reverse := [2]loadpoint.API{a, b}, [2]loadpoint.API{b, a}
	switch diff, band := ga-gb, float64(p.settings.GetPriorityHysteresis()); {
	case diff > band:
		p.leads[key], p.leads[reverse] = true, false
	case diff < -band:
		p.leads[key], p.leads[reverse] = false, true
	}
	return p.leads[key]
}

func (p *Prioritizer) GetChargePowerFlexibility(lp loadpoint.API) float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	var (
		reduceBy float64
		msg      strings.Builder
	)

	for other, power := range p.demand {
		if power <= 0 {
			continue
		}

		if p.outranks(lp, other) {
			reduceBy += power
			msg.WriteString(fmt.Sprintf("%.0fW from %s at prio %d, ", power, other.GetTitle(), other.EffectivePriority()))
		}
	}

	if p.log != nil && reduceBy > 0 {
		p.log.DEBUG.Printf("lp %s at prio %d gets additional %stotal %.0fW\n", lp.GetTitle(), lp.EffectivePriority(), msg.String(), reduceBy)
	}

	return reduceBy
}
