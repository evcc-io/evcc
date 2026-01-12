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

func TestUpdateProfile(t *testing.T) {
	clock := clock.NewMock()

	// adjust for 00:00 in local timezone
	_, o := clock.Now().Zone()
	clock.Add(-time.Duration(o) * time.Second)

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	entity := entity{Name: "foo"}
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
