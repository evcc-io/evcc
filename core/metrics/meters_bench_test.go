package metrics

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
	"github.com/stretchr/testify/require"
)

// BenchmarkMeters measures the meters table access outside the energy page, see BenchmarkHistory,
// and its size: go test -run ^$ -bench Meters ./core/metrics
func BenchmarkMeters(b *testing.B) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.Local)
	benchHistory(b, 16, now)

	var home entity
	require.NoError(b, db.Instance.Where(`"group" = ?`, Home).Order("id DESC").First(&home).Error)

	b.Run("size", func(b *testing.B) {
		var rows, bytes int64
		require.NoError(b, db.Instance.Raw(`SELECT count(*) FROM meters`).Scan(&rows).Error)
		require.NoError(b, db.Instance.Raw(`SELECT sum(pgsize) FROM dbstat WHERE name IN (SELECT name FROM sqlite_master WHERE tbl_name = 'meters')`).Scan(&bytes).Error)
		b.ReportMetric(float64(bytes)/float64(rows), "B/row")
		b.ReportMetric(0, "ns/op")
	})

	// last slot of a meter as the collector reads it on start
	b.Run("last-slot", func(b *testing.B) {
		for b.Loop() {
			var ts int64
			require.NoError(b, db.Instance.Model(new(meter)).Where("meter = ?", home.Id).Select("COALESCE(max(ts), 0)").Scan(&ts).Error)
		}
	})

	// load profiles the optimizer requests for its demand forecast
	b.Run("profile/30d", func(b *testing.B) {
		for b.Loop() {
			_, err := energyProfileFiltered(home, now.AddDate(0, 0, -30), nil, 0.5)
			require.NoError(b, err)
		}
	})

	b.Run("profile/weekday", func(b *testing.B) {
		weekday := int(now.Weekday())
		for b.Loop() {
			_, err := energyProfileFiltered(home, now.AddDate(0, 0, -90), &weekday, 0.5)
			require.NoError(b, err)
		}
	})

	// one slot for one meter as the collector writes it every 15 minutes. Runs last: its slots
	// lie after now and would enter the profiles above.
	// -count repeats the sub-benchmark on the same database, so the slot carries over
	slot := now.Truncate(tariff.SlotDuration)
	b.Run("persist", func(b *testing.B) {
		for b.Loop() {
			require.NoError(b, persist(home, slot, 0.25, 0, nil, false))
			slot = slot.Add(tariff.SlotDuration)
		}
	})
}
