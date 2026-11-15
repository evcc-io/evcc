package metrics

import (
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
	"gorm.io/gorm"
)

// slot origin, stored in meters.recovered
const (
	slotMeasured     = iota
	slotRecovered    // catch-up slot after downtime, excluded from profiles
	slotInterpolated // short gap filled from its neighbours
)

// longer gaps are downtime and stay empty
const maxFillSlots = 4

var slotSeconds = int64(tariff.SlotDuration / time.Second)

// fill persists interpolated slots for the entity's short gaps, as left by a
// restart. A recovered slot after the gap is shared evenly with it instead.
func fill(e entity) error {
	// forecast and temperature are not energy measurements
	if e.Group == Forecast || e.Group == Temperature {
		return nil
	}

	var gaps []struct {
		Ts, PrevTs                   int64 // slot after and before the gap
		Origin                       int
		Energy, ReturnEnergy         float64
		PrevEnergy, PrevReturnEnergy float64
	}
	if err := db.Instance.Raw(`SELECT * FROM (
			SELECT ts, COALESCE(recovered, 0) AS origin,
				COALESCE(energy, 0) AS energy, COALESCE(return_energy, 0) AS return_energy,
				lag(ts) OVER w AS prev_ts,
				COALESCE(lag(energy) OVER w, 0) AS prev_energy,
				COALESCE(lag(return_energy) OVER w, 0) AS prev_return_energy
			FROM meters WHERE meter = ?
			WINDOW w AS (ORDER BY ts)
		) WHERE ts - prev_ts BETWEEN ? AND ?`, e.Id, 2*slotSeconds, (maxFillSlots+1)*slotSeconds).Scan(&gaps).Error; err != nil {
		return err
	}

	for _, g := range gaps {
		n := float64((g.Ts-g.PrevTs)/slotSeconds - 1) // missing slots
		recovered := g.Origin == slotRecovered

		var rows []meter
		for i := 1.0; i <= n; i++ {
			m := meter{
				Meter:        e.Id,
				Timestamp:    g.PrevTs + int64(i)*slotSeconds,
				Energy:       g.PrevEnergy + (g.Energy-g.PrevEnergy)*i/(n+1),
				ReturnEnergy: g.PrevReturnEnergy + (g.ReturnEnergy-g.PrevReturnEnergy)*i/(n+1),
				Recovered:    slotInterpolated,
			}
			if recovered {
				m.Energy = g.Energy / (n + 1)
				m.ReturnEnergy = g.ReturnEnergy / (n + 1)
			}
			rows = append(rows, m)
		}

		if err := db.Instance.Transaction(func(tx *gorm.DB) error {
			if recovered {
				if err := tx.Model(new(meter)).Where("meter = ? AND ts = ?", e.Id, g.Ts).UpdateColumns(map[string]any{
					"energy":        g.Energy / (n + 1),
					"return_energy": g.ReturnEnergy / (n + 1),
				}).Error; err != nil {
					return err
				}
			}
			return tx.Create(rows).Error
		}); err != nil {
			return err
		}
	}

	return nil
}
