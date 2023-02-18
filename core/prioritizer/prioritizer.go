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

func (p *Prioritizer) UpdateChargePowerFlexibility(lp loadpoint.API) {
	if power := lp.GetChargePowerFlexibility(); power >= 0 {
		p.demand[lp] = power
	}
}

func (p *Prioritizer) GetChargePowerFlexibility(lp loadpoint.API) float64 {
	prio := lp.Priority()

	var reduceBy float64
	for lp, power := range p.demand {
		if lp.Priority() < prio {
			reduceBy += power
		}
	}

	return reduceBy
}
