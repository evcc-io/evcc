package metrics

import (
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
)

// EnergyStats holds time-based energy sums in kWh
type EnergyStats struct {
	slot    time.Time // cache key
	Today   float64
	Last24h float64
	Last7d  float64
}

// energyStats sums persisted slot energy for the periods ending at slotStart
func energyStats(entity entity, slotStart, midnight time.Time) (EnergyStats, error) {
	res := EnergyStats{slot: slotStart}

	sqlDB, err := db.Instance.DB()
	if err != nil {
		return res, err
	}

	err = sqlDB.QueryRow(`SELECT
		COALESCE(SUM(CASE WHEN ts >= ? THEN energy END), 0),
		COALESCE(SUM(CASE WHEN ts >= ? THEN energy END), 0),
		COALESCE(SUM(energy), 0)
		FROM meters WHERE meter = ? AND ts >= ? AND ts < ?`,
		midnight.Unix(), slotStart.Add(-24*time.Hour).Unix(),
		entity.Id, slotStart.AddDate(0, 0, -7).Unix(), slotStart.Unix(),
	).Scan(&res.Today, &res.Last24h, &res.Last7d)

	return res, err
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
	res.Today += c.accu.Energy
	return res, nil
}
