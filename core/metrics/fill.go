package metrics

import (
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
)

// slot origin, stored in meters.recovered
const (
	slotMeasured     = iota
	slotRecovered    // catch-up slot after downtime, excluded from profiles
	slotInterpolated // single missing slot filled from its neighbors
)

var slotSeconds = int64(tariff.SlotDuration / time.Second)

// fill persists the mean of its neighbors for a single missing slot before
// the given slot, as left by a restart.
func fill(e entity, slot time.Time) error {
	// temperature slots carry no energy
	if e.Group == Temperature {
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

	// longer gaps are downtime and stay empty, recovered slots hold more than their own energy
	next, prev := ends[0], ends[1]
	if next.Timestamp-prev.Timestamp != 2*slotSeconds || next.Recovered != slotMeasured || prev.Recovered != slotMeasured {
		return nil
	}

	return db.Instance.Create(&meter{
		Meter:        e.Id,
		Timestamp:    prev.Timestamp + slotSeconds,
		Energy:       (prev.Energy + next.Energy) / 2,
		ReturnEnergy: (prev.ReturnEnergy + next.ReturnEnergy) / 2,
		Recovered:    slotInterpolated,
	}).Error
}
