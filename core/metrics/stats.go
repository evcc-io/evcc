package metrics

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
)

// PeriodStats is an entity's metered energy over a period and the Energy page's attribution of it
type PeriodStats struct {
	Energy      float64  `json:"energy"`      // Wh metered
	Autarky     *float64 `json:"autarky"`     // percent of the attributed energy not drawn from the grid
	PricePerKWh *float64 `json:"pricePerKWh"` // average over the priced energy
	Price       *float64 `json:"price"`       // cost of the priced energy
}

// stats derives the period values from the metered energy and the attributed flows
func (e *flowEntity) stats() PeriodStats {
	var sum flowSum
	for _, f := range e.flows {
		sum.energy += f.energy
		sum.cost += f.cost
		sum.priced += f.priced
	}

	res := PeriodStats{Energy: math.Round(e.metered * 1e3)}
	if sum.energy > 0 {
		autarky := 100 * (1 - e.flows[2].energy/sum.energy) // grid is flowSources[2]
		res.Autarky = &autarky
	}
	if sum.priced > 0 {
		price := sum.cost / sum.priced
		res.PricePerKWh = &price
		res.Price = &sum.cost
	}
	return res
}

// periodStats attributes [from,to) like the Energy page and returns the entity's part
func periodStats(entity entity, from, to time.Time) (PeriodStats, error) {
	flow, err := QueryFlow(from, to)
	if err != nil {
		return PeriodStats{}, err
	}
	if e := flow.entities[entity.Id]; e != nil {
		return e.stats(), nil
	}
	return PeriodStats{}, nil
}

// EnergyStats holds time-based energy stats
type EnergyStats struct {
	slot    time.Time // cache key
	Today   float64   // Wh since midnight
	Last24h PeriodStats
	Last7d  PeriodStats
}

// energyStats covers the periods ending at slotStart
func energyStats(entity entity, slotStart, midnight time.Time) (EnergyStats, error) {
	res := EnergyStats{slot: slotStart}

	today, err := periodStats(entity, midnight, slotStart)
	if err != nil {
		return res, err
	}
	res.Today = today.Energy

	if res.Last24h, err = periodStats(entity, slotStart.Add(-24*time.Hour), slotStart); err != nil {
		return res, err
	}
	if res.Last7d, err = periodStats(entity, slotStart.AddDate(0, 0, -7), slotStart); err != nil {
		return res, err
	}
	return res, nil
}

// EnergyStats returns time-based energy sums. Persisted sums are cached per
// slot, the in-flight accumulator is added to today only.
func (c *Collector) EnergyStats() (EnergyStats, error) {
	slotStart := c.accu.clock.Now().Truncate(tariff.SlotDuration)

	if !slotStart.Equal(c.statsCache.slot) {
		stats, err := energyStats(c.entity, slotStart, now.With(c.accu.clock.Now()).BeginningOfDay())
		if err != nil {
			return EnergyStats{}, err
		}

		c.statsCache = stats
	}

	res := c.statsCache
	res.Today = math.Round(res.Today + c.accu.Energy*1e3)
	return res, nil
}
