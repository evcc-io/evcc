package prioritizer

import (
	"github.com/evcc-io/evcc/core/loadpoint"
)

type Prioritizer struct {
	demand map[loadpoint.API]float64
}

func New() *Prioritizer {
	return &Prioritizer{
		demand: make(map[loadpoint.API]float64),
	}
}

func (p *Prioritizer) Consume(lp loadpoint.API) {
	if chargePower := lp.GetChargePower(); chargePower >= 0 {
		p.demand[lp] = chargePower
	}
}

func (p *Prioritizer) Consumable(lp loadpoint.API) float64 {
	prio := lp.Priority()

	var reduceBy float64
	for lp, power := range p.demand {
		if lp.Priority() < prio {
			reduceBy += power
		}
	}

	return reduceBy
}
