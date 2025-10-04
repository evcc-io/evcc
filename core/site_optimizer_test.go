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

// func TestProrateFirstSlot(t *testing.T) {
// 	// Create test hourly profile with known values
// 	// Hour 0: 10, Hour 1: 20, Hour 2: 30, etc.
// 	profile := make([]float64, 24)
// 	for i := range 24 {
// 		profile[i] = float64((i + 1) * 10)
// 	}

// 	tests := []struct {
// 		name              string
// 		now               time.Time
// 		expectedFirstHour float64
// 		expectedLength    int
// 		expectedSecond    float64
// 	}{
// 		{
// 			name:              "start of hour 0 - no proration",
// 			now:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
// 			expectedFirstHour: 10.0, // full hour: no proration applied
// 			expectedLength:    24,   // all 24 hours remain
// 			expectedSecond:    20.0, // hour 1 value
// 		},
// 		{
// 			name:              "30 minutes into hour 0",
// 			now:               time.Date(2024, 1, 1, 0, 30, 0, 0, time.UTC),
// 			expectedFirstHour: 5.0,  // 30min remaining: 10 * 0.5 = 5
// 			expectedLength:    24,   // all 24 hours remain
// 			expectedSecond:    20.0, // hour 1 value unchanged
// 		},
// 		{
// 			name:              "start of hour 2 - no proration",
// 			now:               time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC),
// 			expectedFirstHour: 30.0, // hour 2 value: 30
// 			expectedLength:    22,   // hours 2-23 remain (22 hours)
// 			expectedSecond:    40.0, // hour 3 value
// 		},
// 		{
// 			name:              "15 minutes into hour 2",
// 			now:               time.Date(2024, 1, 1, 2, 15, 0, 0, time.UTC),
// 			expectedFirstHour: 22.5, // 45min remaining: 30 * 0.75 = 22.5
// 			expectedLength:    22,   // hours 2-23 remain (22 hours)
// 			expectedSecond:    40.0, // hour 3 value unchanged
// 		},
// 		{
// 			name:              "45 minutes into hour 5",
// 			now:               time.Date(2024, 1, 1, 5, 45, 0, 0, time.UTC),
// 			expectedFirstHour: 15.0, // 15min remaining: 60 * 0.25 = 15.0
// 			expectedLength:    19,   // hours 5-23 remain (19 hours)
// 			expectedSecond:    70.0, // hour 6 value unchanged
// 		},
// 		{
// 			name:              "10 minutes into hour 10",
// 			now:               time.Date(2024, 1, 1, 10, 10, 0, 0, time.UTC),
// 			expectedFirstHour: 91.67, // 50min remaining: 110 * (50/60) = 91.67
// 			expectedLength:    14,    // hours 10-23 remain (14 hours)
// 			expectedSecond:    120.0, // hour 11 value unchanged
// 		},
// 		{
// 			name:              "near end of day - hour 23",
// 			now:               time.Date(2024, 1, 1, 23, 30, 0, 0, time.UTC),
// 			expectedFirstHour: 120.0, // 30min remaining: 240 * 0.5 = 120
// 			expectedLength:    1,     // only hour 23 remains
// 			expectedSecond:    0.0,   // no second hour
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			result := prorateFirstSlot(tt.now, profile)
// 			require.Equal(t, tt.expectedLength, len(result), "length mismatch")
// 			if len(result) > 0 {
// 				require.InDelta(t, tt.expectedFirstHour, result[0], 0.01, "first hour value mismatch")
// 				// Verify second hour is unchanged (if exists)
// 				if len(result) > 1 {
// 					require.Equal(t, tt.expectedSecond, result[1], "second hour should be unchanged")
// 				}
// 			}
// 		})
// 	}
// }

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
