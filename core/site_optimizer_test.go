package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// func TestCombineSlots(t *testing.T) {
// 	// Create test profile with known values
// 	// Slots 0-3: hour 0, values 1,2,3,4 -> sum = 10
// 	// Slots 4-7: hour 1, values 5,6,7,8 -> sum = 26
// 	// Slots 8-11: hour 2, values 9,10,11,12 -> sum = 42
// 	profile := make([]float64, 96)
// 	for i := range 96 {
// 		profile[i] = float64(i + 1)
// 	}

// 	t.Run("standard profile", func(t *testing.T) {
// 		result := combineSlots(profile)
// 		require.Equal(t, 24, len(result), "should return 24 hours")
// 		require.InDelta(t, 10, result[0], 0.01, "hour 0: slots 0-3 (1+2+3+4)")
// 		require.InDelta(t, 26, result[1], 0.01, "hour 1: slots 4-7 (5+6+7+8)")
// 		require.InDelta(t, 42, result[2], 0.01, "hour 2: slots 8-11 (9+10+11+12)")
// 		require.InDelta(t, 378, result[23], 0.01, "hour 23: slots 92-95 (93+94+95+96)")
// 	})

// 	t.Run("nil profile", func(t *testing.T) {
// 		result := combineSlots(nil)
// 		require.Empty(t, result)
// 	})
// }

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
