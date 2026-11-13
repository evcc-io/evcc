package metrics

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
)

// PeriodStats is an entity's metered energy over a period and the Energy page's attribution of it
type PeriodStats struct {
	Energy        float64  `json:"energy"`        // Wh metered, the rest after solar and battery is grid
	PVEnergy      float64  `json:"pvEnergy"`      // Wh attributed to solar
	BatteryEnergy float64  `json:"batteryEnergy"` // Wh attributed to the battery
	PricePerKWh   *float64 `json:"pricePerKWh"`   // average over the priced energy
	Price         *float64 `json:"price"`         // cost of the priced energy
}

// stats derives the period values from the metered energy and the attributed flows
func (e *flowEntity) stats() PeriodStats {
	var sum flowSum
	for _, f := range e.flows {
		sum.cost += f.cost
		sum.priced += f.priced
	}

	// pv and battery are flowSources[0] and [1]
	res := PeriodStats{
		Energy:        math.Round(e.metered * 1e3),
		PVEnergy:      math.Round(e.flows[0].energy * 1e3),
		BatteryEnergy: math.Round(e.flows[1].energy * 1e3),
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
