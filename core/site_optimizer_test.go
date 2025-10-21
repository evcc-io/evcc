package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/server/db"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSqliteTimestamp(t *testing.T) {
	clock := clock.NewMock()
	clock.Add(time.Hour)

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, metrics.Init())

	metrics.Persist(clock.Now(), 0)

	db, err := db.Instance.DB()
	require.NoError(t, err)

	var (
		ts  metrics.SqlTime
		val float64
	)

	for _, sql := range []string{
		`SELECT ts, val FROM meters`,
		`SELECT min(ts), val FROM meters`,
		`SELECT unixepoch(ts), val FROM meters`,
		`SELECT unixepoch(min(ts)), val FROM meters`,
		`SELECT min(ts) AS ts, avg(val) AS val
			FROM meters
			GROUP BY strftime("%H:%M", ts)
			ORDER BY ts`,
	} {
		require.NoError(t, db.QueryRow(sql).Scan(&ts, &val))
		require.True(t, clock.Now().Equal(time.Time(ts)), "expected %v, got %v", clock.Now().Local(), time.Time(ts).Local())
	}

	require.NoError(t, db.QueryRow(`SELECT ts, val FROM meters WHERE ts >= ?`, clock.Now()).Scan(&ts, &val))
	require.True(t, clock.Now().Equal(time.Time(ts)), "expected %v, got %v", clock.Now().Local(), time.Time(ts).Local())
}

func TestUpdateHouseholdProfile(t *testing.T) {
	clock := clock.NewMock()

	// make sure test data added starting 00:00 local time
	clock.Set(now.With(clock.Now()).BeginningOfDay())

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	metrics.Init()

	// 2 days of data
	// day 1:   0 ...  95
	// day 2:  96 ... 181
	for i := range 4 * 2 * 24 {
		metrics.Persist(clock.Now(), float64(i))
		clock.Add(15 * time.Minute)
	}

	{
		from := clock.Now().Local().AddDate(0, 0, -2).Add(12 * time.Hour) // 12:00 of day 0

		prof, err := metrics.Profile(from)
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

		prof, err := metrics.Profile(from)
		require.NoError(t, err)

		var expected [96]float64
		for i := range expected {
			expected[i] = float64(0+96+2*i) / 2
		}

		require.Equal(t, expected, *prof, "full profile: expected %v, got %v", expected, *prof)
	}
}

func TestLoadpointProfile(t *testing.T) {
	ctrl := gomock.NewController(t)

	lp := loadpoint.NewMockAPI(ctrl)
	lp.EXPECT().GetMode().Return(api.ModeMinPV).AnyTimes()
	lp.EXPECT().GetStatus().Return(api.StatusC).AnyTimes()
	lp.EXPECT().GetChargePower().Return(10000.0).AnyTimes()   //  10 kW
	lp.EXPECT().EffectiveMinPower().Return(1000.0).AnyTimes() //   1 kW
	lp.EXPECT().GetRemainingEnergy().Return(1.8).AnyTimes()   // 1.8 kWh

	// expected slots: 0.25 kWh...
	require.Equal(t, []float64{250, 250, 250, 250, 250, 250, 250, 50}, loadpointProfile(lp, 8))
}
