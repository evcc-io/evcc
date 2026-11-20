package metrics

import (
	"math"
	"slices"
	"time"

	"github.com/evcc-io/evcc/db"
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

// dailyPeakEnergy returns the max persisted slot energy in kWh per day of the month of
// slotStart, nil for days without a slot. Queried by group so a replaced meter keeps the month.
func dailyPeakEnergy(group string, slotStart time.Time) ([]*float64, error) {
	type row struct {
		Day    int
		Energy float64
	}

	var rows []row
	err := db.Instance.Table("meters m").
		Select(`CAST(strftime('%d', m.ts, 'unixepoch', 'localtime') AS INTEGER) AS day, MAX(m.energy) AS energy`).
		Joins("JOIN entities e ON m.meter = e.id").
		Where(`e."group" = ? AND m.ts >= ? AND m.ts < ? AND COALESCE(m.recovered, 0) = 0`,
			group, now.With(slotStart).BeginningOfMonth().Unix(), slotStart.Unix()).
		Group("day").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	days := make([]*float64, slotStart.Day())
	for _, r := range rows {
		if r.Day >= 1 && r.Day <= len(days) {
			days[r.Day-1] = &r.Energy
		}
	}

	return days, nil
}

// DailyPeakEnergy returns the month's max slot import in kWh per day, index 0 is the 1st,
// nil without a complete slot. Persisted peaks are cached per slot, the in-flight slot raises today.
func (c *Collector) DailyPeakEnergy() ([]*float64, error) {
	slotStart := c.accu.clock.Now().Truncate(tariff.SlotDuration)

	if !slotStart.Equal(c.peakSlot) {
		days, err := dailyPeakEnergy(c.entity.Group, slotStart)
		if err != nil {
			return nil, err
		}

		c.peakSlot, c.peakDays = slotStart, days
	}

	res := slices.Clone(c.peakDays)

	// downtime catchup energy is not a peak, and zero in-flight energy is not a measurement
	if today, energy := len(res)-1, c.accu.Energy; energy > 0 && !c.recovering() && (res[today] == nil || energy > *res[today]) {
		res[today] = &energy
	}

	return res, nil
}
