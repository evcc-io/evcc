package metrics

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/db"
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
