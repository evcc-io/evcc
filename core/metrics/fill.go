package metrics

import (
	"database/sql"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/tariff"
)

// slot origin, stored in meters.recovered
const (
	slotMeasured     = iota
	slotRecovered    // catch-up slot after downtime, excluded from profiles
	slotInterpolated // single missing slot filled from its neighbors
)

var slotSeconds = int64(tariff.SlotDuration / time.Second)

// fillVersion is the version of the interpolation that history was backfilled with
const fillVersion = 1

// fillGaps interpolates a single missing slot between two measured slots.
// Longer gaps are downtime and stay empty, recovered slots hold more than
// their own energy, temperature slots carry no energy.
const fillGaps = `INSERT INTO meters (meter, ts, energy, return_energy, recovered)
	SELECT n.meter, n.ts - @slot,
		(COALESCE(p.energy, 0) + COALESCE(n.energy, 0)) / 2.0,
		(COALESCE(p.return_energy, 0) + COALESCE(n.return_energy, 0)) / 2.0,
		@interpolated
	FROM meters n
	JOIN meters p ON p.meter = n.meter AND p.ts = n.ts - 2 * @slot
	JOIN entities e ON e.id = n.meter
	WHERE e."group" <> @temperature
		AND COALESCE(n.recovered, 0) = @measured AND COALESCE(p.recovered, 0) = @measured
		AND NOT EXISTS (SELECT 1 FROM meters g WHERE g.meter = n.meter AND g.ts = n.ts - @slot)`

// interpolate fills gaps, optionally narrowed by a condition on the newer neighbor n
func interpolate(where string, args ...any) error {
	return db.Instance.Exec(fillGaps+where, append(args, map[string]any{
		"slot":         slotSeconds,
		"measured":     slotMeasured,
		"interpolated": slotInterpolated,
		"temperature":  Temperature,
	})...).Error
}

// fill interpolates a single missing slot before the given slot, as left by a restart.
func fill(e entity, slot time.Time) error {
	return interpolate(" AND n.meter = @meter AND n.ts = @ts", sql.Named("meter", e.Id), sql.Named("ts", slot.Unix()))
}

// backfill interpolates the gaps of the existing history once per fill version
func backfill() error {
	if v, _ := settings.Int(keys.MetricsFillVersion); v >= fillVersion {
		return nil
	}

	if err := interpolate(""); err != nil {
		return err
	}

	// the rolled up days do not hold the filled slots
	if err := resetDaily(nil); err != nil {
		return err
	}

	settings.SetInt(keys.MetricsFillVersion, fillVersion)

	return nil
}
