package prioritizer

import (
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

type Prioritizer struct {
	mu     sync.Mutex
	log    *util.Logger
	demand map[loadpoint.API]float64
}

func New(log *util.Logger) *Prioritizer {
	return &Prioritizer{
		log:    log,
		demand: make(map[loadpoint.API]float64),
	}
}

func (p *Prioritizer) UpdateChargePowerFlexibility(lp loadpoint.API) {
	if power := lp.GetChargePowerFlexibility(); power >= 0 {
		p.mu.Lock()
		p.demand[lp] = power
		p.mu.Unlock()
	}
}

func (p *Prioritizer) GetChargePowerFlexibility(lp loadpoint.API) float64 {
	prio := lp.EffectivePriority()

	var (
		reduceBy float64
		msg      string
	)

	for lp, power := range p.demand {
		if lp.EffectivePriority() < prio && power > 0 {
			reduceBy += power
			msg += fmt.Sprintf("%.0fW from %s at prio %d, ", power, lp.Title(), lp.EffectivePriority())
		}
	}

	if p.log != nil && reduceBy > 0 {
		p.log.DEBUG.Printf("lp %s at prio %d gets additional %stotal %.0fW\n", lp.Title(), lp.EffectivePriority(), msg, reduceBy)
	}

	return reduceBy
}
