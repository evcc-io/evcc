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

// fill persists interpolated slots for a short gap before the given slot, as
// left by a restart. A recovered slot is shared evenly with the gap instead.
func fill(e entity, slot time.Time) error {
	// forecast and temperature are not energy measurements
	if e.Group == Forecast || e.Group == Temperature {
		return nil
	}

	// the slot and its predecessor
	var ends []meter
	if err := db.Instance.Where("meter = ? AND ts <= ?", e.Id, slot.Unix()).Order("ts DESC").Limit(2).Find(&ends).Error; err != nil {
		return err
	}
	if len(ends) < 2 {
		return nil // first slot of the entity, no predecessor
	}

	next, prev := ends[0], ends[1]
	missing := (next.Timestamp-prev.Timestamp)/slotSeconds - 1
	if missing < 1 || missing > maxFillSlots {
		return nil
	}

	n := float64(missing + 1)
	recovered := next.Recovered == slotRecovered

	rows := make([]meter, missing)
	for i := range rows {
		f := float64(i+1) / n
		rows[i] = meter{
			Meter:        e.Id,
			Timestamp:    prev.Timestamp + int64(i+1)*slotSeconds,
			Energy:       prev.Energy + (next.Energy-prev.Energy)*f,
			ReturnEnergy: prev.ReturnEnergy + (next.ReturnEnergy-prev.ReturnEnergy)*f,
			Recovered:    slotInterpolated,
		}
		if recovered {
			rows[i].Energy = next.Energy / n
			rows[i].ReturnEnergy = next.ReturnEnergy / n
		}
	}

	return db.Instance.Transaction(func(tx *gorm.DB) error {
		if recovered {
			if err := tx.Model(new(meter)).Where("meter = ? AND ts = ?", e.Id, next.Timestamp).UpdateColumns(map[string]any{
				"energy":        next.Energy / n,
				"return_energy": next.ReturnEnergy / n,
			}).Error; err != nil {
				return err
			}
		}
		return tx.Create(rows).Error
	})
}
