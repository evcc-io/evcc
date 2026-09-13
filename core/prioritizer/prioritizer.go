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

type rankingConfig struct {
	strategy   api.PriorityStrategy
	basis      api.PriorityBasis
	hysteresis int
}

type leadState struct {
	leads    bool
	vehicles [2]api.Vehicle
}

type Prioritizer struct {
	mu         sync.Mutex
	log        *util.Logger
	settings   Settings
	demand     map[loadpoint.API]float64
	leads      map[[2]loadpoint.API]leadState
	config     rankingConfig
	configured bool
}

func New(log *util.Logger, settings Settings) *Prioritizer {
	return &Prioritizer{
		log:      log,
		settings: settings,
		demand:   make(map[loadpoint.API]float64),
		leads:    make(map[[2]loadpoint.API]leadState),
	}
}

func (p *Prioritizer) UpdateChargePowerFlexibility(lp loadpoint.API, rates api.Rates) {
	if power := lp.GetChargePowerFlexibility(rates); power >= 0 {
		p.mu.Lock()
		p.demand[lp] = power
		if status := lp.GetStatus(); status != api.StatusB && status != api.StatusC {
			p.clearLeads(lp)
		}
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
	config := rankingConfig{
		strategy:   p.settings.GetPriorityStrategy(),
		basis:      p.settings.GetPriorityBasis(),
		hysteresis: p.settings.GetPriorityHysteresis(),
	}
	if !p.configured || config != p.config {
		clear(p.leads)
		p.config, p.configured = config, true
	}

	if pa, pb := a.EffectivePriority(), b.EffectivePriority(); pa != pb {
		return pa > pb
	}

	ga, oka := a.PriorityGap(config.strategy, config.basis)
	gb, okb := b.PriorityGap(config.strategy, config.basis)
	key, reverse := [2]loadpoint.API{a, b}, [2]loadpoint.API{b, a}
	if !oka || !okb {
		delete(p.leads, key)
		delete(p.leads, reverse)
		return false
	}

	vehicles := [2]api.Vehicle{a.GetVehicle(), b.GetVehicle()}
	if state, ok := p.leads[key]; ok && state.vehicles != vehicles {
		delete(p.leads, key)
		delete(p.leads, reverse)
	}

	switch diff, band := ga-gb, float64(config.hysteresis); {
	case diff > band:
		p.leads[key] = leadState{true, vehicles}
		p.leads[reverse] = leadState{false, [2]api.Vehicle{vehicles[1], vehicles[0]}}
	case diff < -band:
		p.leads[key] = leadState{false, vehicles}
		p.leads[reverse] = leadState{true, [2]api.Vehicle{vehicles[1], vehicles[0]}}
	}
	return p.leads[key].leads
}

func (p *Prioritizer) clearLeads(lp loadpoint.API) {
	for pair := range p.leads {
		if pair[0] == lp || pair[1] == lp {
			delete(p.leads, pair)
		}
	}
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
