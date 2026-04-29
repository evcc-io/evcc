package metrics

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/require"
)

func TestSqliteTimestamp(t *testing.T) {
	clock := clock.NewMock()
	clock.Add(time.Hour)

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	entity := entity{Name: "foo"}
	require.NoError(t, db.Instance.FirstOrCreate(&entity).Error)

	persist(entity, clock.Now(), 0, 0)

	db, err := db.Instance.DB()
	require.NoError(t, err)

	var (
		ts  SqlTime
		val float64
	)

	for _, sql := range []string{
		`SELECT ts, import FROM meters`,
		`SELECT min(ts), import FROM meters`,
		`SELECT unixepoch(ts), import FROM meters`,
		`SELECT unixepoch(min(ts)), import FROM meters`,
		`SELECT min(ts) AS ts, avg(import) AS import
			FROM meters
			GROUP BY strftime("%H:%M", ts)
			ORDER BY ts`,
	} {
		require.NoError(t, db.QueryRow(sql).Scan(&ts, &val))
		require.True(t, clock.Now().Equal(time.Time(ts)), "expected %v, got %v", clock.Now().Local(), time.Time(ts).Local())
	}

	require.NoError(t, db.QueryRow(`SELECT ts, import FROM meters WHERE ts >= ?`, clock.Now()).Scan(&ts, &val))
	require.True(t, clock.Now().Equal(time.Time(ts)), "expected %v, got %v", clock.Now().Local(), time.Time(ts).Local())
}

func TestQueryImportEnergyUTCFilter(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	e := entity{Name: "grid", Group: "grid"}
	require.NoError(t, db.Instance.FirstOrCreate(&e).Error)

	// insert 4 slots at 16:00, 16:15, 16:30, 16:45 local time
	loc := time.Now().Location()
	base := time.Date(2026, 4, 15, 16, 0, 0, 0, loc)

	for i := range 4 {
		ts := base.Add(time.Duration(i) * 15 * time.Minute)
		require.NoError(t, persist(e, ts, 0, float64(i+1)))
	}

	// query with UTC times that cover all 4 slots
	// base is 16:00 local, convert to UTC and use a from before and to after
	from := base.Add(-time.Hour).UTC()
	to := base.Add(time.Hour).UTC()

	res, err := QueryImportEnergy(from, to, "15m", false)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Len(t, res[0].Data, 4, "expected all 4 slots, got %d", len(res[0].Data))

	var totalExport float64
	for _, s := range res[0].Data {
		totalExport += s.Export
	}
	require.InDelta(t, 1+2+3+4, totalExport, 0.001)
}

func TestQueryImportEnergyGrouped(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	// two entities sharing the same group, different names
	e1 := entity{Id: 2, Name: "db:12", Group: "grid"}
	require.NoError(t, db.Instance.Create(&e1).Error)
	e2 := entity{Id: 3, Name: "db:13", Group: "grid"}
	require.NoError(t, db.Instance.Create(&e2).Error)

	loc := time.Now().Location()
	base := time.Date(2026, 4, 15, 16, 0, 0, 0, loc)

	require.NoError(t, persist(e1, base, 1, 0))
	require.NoError(t, persist(e2, base, 2, 0))
	require.NoError(t, persist(e1, base.Add(15*time.Minute), 3, 0))
	require.NoError(t, persist(e2, base.Add(15*time.Minute), 4, 0))

	from := base.Add(-time.Hour).UTC()
	to := base.Add(time.Hour).UTC()

	// ungrouped: 2 series
	res, err := QueryImportEnergy(from, to, "15m", false)
	require.NoError(t, err)
	require.Len(t, res, 2)

	// grouped: 1 series, values summed per bucket
	res, err = QueryImportEnergy(from, to, "15m", true)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, "grid", res[0].Group)
	require.Empty(t, res[0].Name)
	require.Len(t, res[0].Data, 2)
	require.InDelta(t, 1+2, res[0].Data[0].Import, 0.001)
	require.InDelta(t, 3+4, res[0].Data[1].Import, 0.001)
}

func TestUpdateProfile(t *testing.T) {
	clock := clock.NewMock()

	// adjust for 00:00 in local timezone
	_, o := clock.Now().Zone()
	clock.Add(-time.Duration(o) * time.Second)

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	entity := entity{Id: 2, Name: "foo"}
	require.NoError(t, db.Instance.FirstOrCreate(&entity).Error)

	// 2 days of data
	// day 1:   0 ...  95
	// day 2:  96 ... 181
	for i := range 4 * 2 * 24 {
		persist(entity, clock.Now(), float64(i), float64(i))
		clock.Add(15 * time.Minute)
	}

	{
		from := clock.Now().Local().AddDate(0, 0, -2).Add(12 * time.Hour) // 12:00 of day 0

		prof, err := importProfile(entity, from)
		require.NoError(t, err)

		var expected [96]float64
		for i := range expected {
			if i < 48 {
				expected[i] = float64(48+i+144+i) / 2
				continue
			}
			expected[i] = float64(96 - 48 + i)
		}

		require.Equal(t, expected, *prof, "partial profile: expected %v, got %v", expected, *prof)
	}

	{
		from := clock.Now().Local().AddDate(0, 0, -3).Add(12 * time.Hour) // 12:00 of day -1

		prof, err := importProfile(entity, from)
		require.NoError(t, err)

		var expected [96]float64
		for i := range expected {
			expected[i] = float64(0+96+2*i) / 2
		}

		require.Equal(t, expected, *prof, "full profile: expected %v, got %v", expected, *prof)
	}
}
