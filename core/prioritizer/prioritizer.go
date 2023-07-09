package prioritizer

import (
	"fmt"

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
	prio := lp.GetPriority()

	msg := fmt.Sprintf("lp %s at prio %d gets additional ", lp.Title(), lp.GetPriority())

	var reduceBy float64
	for lp, power := range p.demand {
		if lp.GetPriority() < prio {
			reduceBy += power
			msg += fmt.Sprintf("%.0f from %s at prio %d, ", power, lp.Title(), lp.GetPriority())
		}
	}

	msg += fmt.Sprintf("total %.0f", reduceBy)
	fmt.Println(">", msg)

	return reduceBy
}
