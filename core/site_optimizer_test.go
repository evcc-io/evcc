package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/server/db"
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

func TestSlotsToHours(t *testing.T) {
	// Create a test profile with known values
	// Slots 0-3: hour 0 (00:00-01:00), values 1,2,3,4 -> sum = 10
	// Slots 4-7: hour 1 (01:00-02:00), values 5,6,7,8 -> sum = 26
	// etc.
	var profile [96]float64
	for i := range 96 {
		profile[i] = float64(i + 1)
	}

	tests := []struct {
		name     string
		now      time.Time
		expected []float32
	}{
		{
			name: "start of hour",
			now:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // 00:00
			expected: []float32{
				10, // hour 0: slots 0-3 (1+2+3+4)
				26, // hour 1: slots 4-7 (5+6+7+8)
				42, // hour 2: slots 8-11 (9+10+11+12)
			},
		},
		{
			name: "middle of hour",
			now:  time.Date(2024, 1, 1, 0, 30, 0, 0, time.UTC), // 00:30
			expected: []float32{
				3.5, // remaining 30min of hour 0: slots 2-3 (3+4) * 0.5 = 7 * 0.5 = 3.5
				26,  // hour 1: slots 4-7 (5+6+7+8)
				42,  // hour 2: slots 8-11 (9+10+11+12)
			},
		},
		{
			name: "quarter past hour",
			now:  time.Date(2024, 1, 1, 0, 15, 0, 0, time.UTC), // 00:15
			expected: []float32{
				6.75, // remaining 45min of hour 0: slots 1-3 (2+3+4) * 0.75 = 9 * 0.75 = 6.75
				26,   // hour 1: slots 4-7 (5+6+7+8)
				42,   // hour 2: slots 8-11 (9+10+11+12)
			},
		},
		{
			name: "middle of slot early",
			now:  time.Date(2024, 1, 1, 0, 10, 0, 0, time.UTC), // 00:10 (middle of slot 0)
			expected: []float32{
				8.33, // remaining 50min of hour 0: slots 0-3 (1+2+3+4) * (50/60) = 10 * 0.833 = 8.33
				26,   // hour 1: slots 4-7 (5+6+7+8)
				42,   // hour 2: slots 8-11 (9+10+11+12)
			},
		},
		{
			name: "middle of slot late",
			now:  time.Date(2024, 1, 1, 0, 50, 0, 0, time.UTC), // 00:50 (middle of slot 3)
			expected: []float32{
				0.67, // remaining 10min of hour 0: slot 3 (4) * (10/60) = 4 * 0.167 = 0.67
				26,   // hour 1: slots 4-7 (5+6+7+8)
				42,   // hour 2: slots 8-11 (9+10+11+12)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slotsToHours(tt.now, &profile)
			require.GreaterOrEqual(t, len(result), len(tt.expected), "insufficient result length")

			for i, expected := range tt.expected {
				require.InDelta(t, expected, result[i], 0.01, "mismatch at index %d", i)
			}
		})
	}
}

func TestSlotsToHoursEdgeCases(t *testing.T) {
	var profile [96]float64
	for i := range 96 {
		profile[i] = float64(i + 1)
	}

	t.Run("nil profile", func(t *testing.T) {
		result := slotsToHours(time.Now(), nil)
		require.Empty(t, result)
	})

	t.Run("end of day", func(t *testing.T) {
		now := time.Date(2024, 1, 1, 23, 45, 0, 0, time.UTC) // 23:45
		result := slotsToHours(now, &profile)

		// Should only return the remaining 15min of the current hour
		expected := []float32{24} // slot 95 (value 96) * 0.25 (15min/60min)
		require.Equal(t, len(expected), len(result))
		require.InDelta(t, expected[0], result[0], 0.01)
	})

	t.Run("exact hour boundary", func(t *testing.T) {
		now := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC) // 01:00
		result := slotsToHours(now, &profile)

		// Should start from hour 1 (slots 4-7)
		require.GreaterOrEqual(t, len(result), 1)
		require.InDelta(t, float32(26), result[0], 0.01) // 5+6+7+8 = 26
	})
}

func TestLoadpointProfile(t *testing.T) {
	ctrl := gomock.NewController(t)

	lp := loadpoint.NewMockAPI(ctrl)
	lp.EXPECT().GetMode().Return(api.ModeMinPV).AnyTimes()
	lp.EXPECT().GetStatus().Return(api.StatusC).AnyTimes()
	lp.EXPECT().GetChargePower().Return(10000.0).AnyTimes()   // 1 0kW
	lp.EXPECT().EffectiveMinPower().Return(1000.0).AnyTimes() // 1 kW
	lp.EXPECT().GetRemainingEnergy().Return(2.0).AnyTimes()   // 2 kWh

	// expected slots: 0.25/ 1.0 / 0.75 kWh
	require.Equal(t, []float64{250, 1000, 750}, loadpointProfile(lp, 15*time.Minute, 3))
}
