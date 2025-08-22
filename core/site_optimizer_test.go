package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCombineSlots(t *testing.T) {
	// Create test profile with known values
	// Slots 0-3: hour 0, values 1,2,3,4 -> sum = 10
	// Slots 4-7: hour 1, values 5,6,7,8 -> sum = 26
	// Slots 8-11: hour 2, values 9,10,11,12 -> sum = 42
	profile := make([]float64, 96)
	for i := range 96 {
		profile[i] = float64(i + 1)
	}

	t.Run("standard profile", func(t *testing.T) {
		result := combineSlots(profile)
		require.Equal(t, 24, len(result), "should return 24 hours")
		require.InDelta(t, 10, result[0], 0.01, "hour 0: slots 0-3 (1+2+3+4)")
		require.InDelta(t, 26, result[1], 0.01, "hour 1: slots 4-7 (5+6+7+8)")
		require.InDelta(t, 42, result[2], 0.01, "hour 2: slots 8-11 (9+10+11+12)")
		require.InDelta(t, 378, result[23], 0.01, "hour 23: slots 92-95 (93+94+95+96)")
	})

	t.Run("nil profile", func(t *testing.T) {
		result := combineSlots(nil)
		require.Empty(t, result)
	})
}

func TestProrateFirstHour(t *testing.T) {
	// Create test hourly profile with known values
	// Hour 0: 10, Hour 1: 20, Hour 2: 30, etc.
	profile := make([]float64, 24)
	for i := range 24 {
		profile[i] = float64((i + 1) * 10)
	}

	tests := []struct {
		name              string
		now               time.Time
		expectedFirstHour float64
		expectedLength    int
		expectedSecond    float64
	}{
		{
			name:              "start of hour 0 - no proration",
			now:               time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedFirstHour: 10.0, // full hour: no proration applied
			expectedLength:    24,   // all 24 hours remain
			expectedSecond:    20.0, // hour 1 value
		},
		{
			name:              "30 minutes into hour 0",
			now:               time.Date(2024, 1, 1, 0, 30, 0, 0, time.UTC),
			expectedFirstHour: 5.0,  // 30min remaining: 10 * 0.5 = 5
			expectedLength:    24,   // all 24 hours remain
			expectedSecond:    20.0, // hour 1 value unchanged
		},
		{
			name:              "start of hour 2 - no proration",
			now:               time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC),
			expectedFirstHour: 30.0, // hour 2 value: 30
			expectedLength:    22,   // hours 2-23 remain (22 hours)
			expectedSecond:    40.0, // hour 3 value
		},
		{
			name:              "15 minutes into hour 2",
			now:               time.Date(2024, 1, 1, 2, 15, 0, 0, time.UTC),
			expectedFirstHour: 22.5, // 45min remaining: 30 * 0.75 = 22.5
			expectedLength:    22,   // hours 2-23 remain (22 hours)
			expectedSecond:    40.0, // hour 3 value unchanged
		},
		{
			name:              "45 minutes into hour 5",
			now:               time.Date(2024, 1, 1, 5, 45, 0, 0, time.UTC),
			expectedFirstHour: 15.0, // 15min remaining: 60 * 0.25 = 15.0
			expectedLength:    19,   // hours 5-23 remain (19 hours)
			expectedSecond:    70.0, // hour 6 value unchanged
		},
		{
			name:              "10 minutes into hour 10",
			now:               time.Date(2024, 1, 1, 10, 10, 0, 0, time.UTC),
			expectedFirstHour: 91.67, // 50min remaining: 110 * (50/60) = 91.67
			expectedLength:    14,    // hours 10-23 remain (14 hours)
			expectedSecond:    120.0, // hour 11 value unchanged
		},
		{
			name:              "near end of day - hour 23",
			now:               time.Date(2024, 1, 1, 23, 30, 0, 0, time.UTC),
			expectedFirstHour: 120.0, // 30min remaining: 240 * 0.5 = 120
			expectedLength:    1,     // only hour 23 remains
			expectedSecond:    0.0,   // no second hour
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prorateFirstHour(tt.now, profile)
			require.Equal(t, tt.expectedLength, len(result), "length mismatch")
			if len(result) > 0 {
				require.InDelta(t, tt.expectedFirstHour, result[0], 0.01, "first hour value mismatch")
				// Verify second hour is unchanged (if exists)
				if len(result) > 1 {
					require.Equal(t, tt.expectedSecond, result[1], "second hour should be unchanged")
				}
			}
		})
	}
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
