package metrics

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
	"github.com/stretchr/testify/require"
)

// benchHistory creates a database with every slot of the 2 years before now filled
func benchHistory(b *testing.B, entities int, now time.Time) {
	require.NoError(b, db.NewInstance("sqlite", filepath.Join(b.TempDir(), "evcc.db")))
	require.NoError(b, SetupSchema())

	groups := []string{Home, Grid, PV, Battery}
	var ids []int
	for i := range entities {
		group := Consumer
		switch {
		case i < len(groups):
			group = groups[i]
		case i%3 == 0:
			group = Loadpoint
		}
		e := entity{Name: fmt.Sprintf("%s-%d", group, i), Group: group}
		require.NoError(b, db.Instance.Create(&e).Error)
		ids = append(ids, e.Id)
	}

	sqlDB, err := db.Instance.DB()
	require.NoError(b, err)
	tx, err := sqlDB.Begin()
	require.NoError(b, err)
	slot, err := tx.Prepare("INSERT INTO meters (meter, ts, energy, return_energy) VALUES (?, ?, ?, ?)")
	require.NoError(b, err)
	defer slot.Close()
	price, err := tx.Prepare("INSERT INTO tariffs (ts, grid, feedin, co2) VALUES (?, ?, ?, ?)")
	require.NoError(b, err)
	defer price.Close()

	end := now.Truncate(tariff.SlotDuration)
	for ts := dayStart(now.AddDate(-2, 0, 0)); ts.Before(end); ts = ts.Add(tariff.SlotDuration) {
		for i, id := range ids {
			_, err := slot.Exec(id, ts.Unix(), 0.25+float64(i)/64, float64(i%3)/16)
			require.NoError(b, err)
		}
		_, err := price.Exec(ts.Unix(), 0.25+float64(ts.Hour())/100, 0.08, 400.0)
		require.NoError(b, err)
	}
	require.NoError(b, tx.Commit())
}

// BenchmarkHistory measures the requests of the energy page, read from the
// slots and from the rollup: go test -run ^$ -bench History ./core/metrics
func BenchmarkHistory(b *testing.B) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.Local)
	yesterday := dayStart(now.AddDate(0, 0, -1))

	queries := []struct {
		name string
		run  func() error
	}{
		{"energy/year-by-month", func() error {
			_, err := QueryEnergy(now.AddDate(-1, 0, 0), now, "month", false)
			return err
		}},
		{"energy/month-by-day", func() error {
			_, err := QueryEnergy(now.AddDate(0, -1, 0), now, "day", false)
			return err
		}},
		{"energy/day-by-15m", func() error {
			_, err := QueryEnergy(yesterday, nextDay(yesterday), "15m", false)
			return err
		}},
		{"flow/year", func() error {
			_, err := QueryFlow(now.AddDate(-1, 0, 0), now)
			return err
		}},
		{"tariffs/year-by-month", func() error {
			_, err := QueryTariffs(now.AddDate(-1, 0, 0), now, "month")
			return err
		}},
	}

	for _, entities := range []int{16, 32} {
		benchHistory(b, entities, now)

		runQueries := func(state string) {
			for _, q := range queries {
				b.Run(fmt.Sprintf("entities=%d/%s/%s", entities, state, q.name), func(b *testing.B) {
					for b.Loop() {
						require.NoError(b, q.run())
					}
				})
			}
		}

		runQueries("slots")

		// rolls up once, so it is timed by hand
		start := time.Now()
		require.NoError(b, rollupDays(now))
		initial := time.Since(start).Seconds()

		b.Run(fmt.Sprintf("entities=%d/rollup", entities), func(b *testing.B) {
			b.ReportMetric(initial, "s/rollup")
			b.ReportMetric(0, "ns/op")
		})

		runQueries("rollup")

		// a delete a year ago, the delete itself is not timed
		old := dayStart(now.AddDate(-1, 0, 0))
		deleteHour := func(b *testing.B) {
			_, err := DeleteEnergy(old, old.Add(time.Hour))
			require.NoError(b, err)
		}

		// the requests until the rollup has caught up
		deleteHour(b)
		runQueries("deleted")

		b.Run(fmt.Sprintf("entities=%d/deleted/rollup", entities), func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				deleteHour(b)
				b.StartTimer()
				require.NoError(b, rollupDays(now))
			}
		})
	}
}
