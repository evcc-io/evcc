package core

import (
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type Prioritizer struct {
	demand map[int]float64
}

func (p *Prioritizer) Prioritize(prio int, consumablePower float64) float64 {
	p.demand[prio] = consumablePower

	keys := maps.Keys(p.demand)
	slices.Sort(keys)

	var reduceBy float64
	for _, k := range keys {
		if k > prio {
			reduceBy += p.demand[k]
		}
	}

	return reduceBy
}
