package core

import (
	"slices"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestApplyTemperatureCorrection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now().Truncate(15 * time.Minute)

	mockTariff := api.NewMockTariff(ctrl)

	rates := []api.Rate{}

	// Past 7 days - create slots for ALL hours to ensure historical data exists
	for day := -7; day < 0; day++ {
		for hour := range 24 {
			for slot := range 4 {
				baseTime := now.AddDate(0, 0, day).Truncate(24 * time.Hour)
				rates = append(rates, api.Rate{
					Start: baseTime.Add(time.Duration(hour)*time.Hour + time.Duration(slot)*15*time.Minute),
					End:   baseTime.Add(time.Duration(hour)*time.Hour + time.Duration(slot+1)*15*time.Minute),
					Value: 10.0,
				})
			}
		}
	}

	// Future forecast: 12 slots (3 hours)
	// First hour: 5°C, second hour: 15°C, third hour: 20°C
	for i := range 12 {
		temp := 5.0
		switch {
		case i >= 8:
			temp = 20.0
		case i >= 4:
			temp = 15.0
		}
		rates = append(rates, api.Rate{
			Start: now.Add(time.Duration(i) * 15 * time.Minute),
			End:   now.Add(time.Duration(i+1) * 15 * time.Minute),
			Value: temp,
		})
	}

	mockTariff.EXPECT().Rates().Return(rates, nil).AnyTimes()

	site := &Site{
		log:     util.NewLogger("test"),
		tariffs: &tariff.Tariffs{Temperature: mockTariff},
	}

	profile := slices.Repeat([]float64{2.0}, 12)

	// 1. Linear scaling (binned == nil)
	result := site.applyTemperatureCorrection("test", profile, nil)

	require.Len(t, result, 12)

	// Verify correction is applied: first hour should increase, second hour should decrease
	assert.Greater(t, result[0], 2.0, "first hour should increase (colder forecast)")
	assert.Less(t, result[4], 2.0, "second hour should decrease (warmer forecast)")
	assert.Greater(t, result[0], result[4], "first hour should be higher than second hour")
	// warm forecast scales down naturally and clamps to minCorrection (10%)
	assert.InDelta(t, 0.2, result[8], 1e-6, "third hour scales to minCorrection floor")

	// 2. Temperature-binned lookup: same bins for every slot of the day, so the
	// result depends on the forecast temperature only
	bins := map[int]float64{
		5:  3.5, // 5°C forecast
		10: 2.2,
		12: 1.0,
		20: 0.8, // 20°C forecast, recorded DHW cycle
	}
	binned := make(map[int]map[int]float64)
	for slot := range 96 {
		binned[slot] = bins
	}
	resultBinned := site.applyTemperatureCorrection("test", profile, binned)
	assert.InDelta(t, 3.5, resultBinned[0], 1e-6, "binned exact match should be used directly")
	assert.Less(t, resultBinned[4], 2.0, "15°C without bin falls back to scaling")
	assert.InDelta(t, 0.8, resultBinned[8], 1e-6, "binned exact match for warm temp (>=18°C) should be used directly")
}

func TestLookupBin(t *testing.T) {
	bins := map[int]float64{
		5:  3.0,
		7:  5.0,
		10: 2.0,
		12: 4.0,
	}

	// Exact match
	val, ok := lookupBin(bins, 5.2)
	assert.True(t, ok)
	assert.InDelta(t, 3.0, val, 1e-6)

	// Adjacent ±1 interpolation (e.g. 11°C interpolates between 10°C and 12°C)
	val, ok = lookupBin(bins, 11.0)
	assert.True(t, ok)
	assert.InDelta(t, 3.0, val, 1e-6) // (2.0 + 4.0) / 2

	// Adjacent single match (e.g. 6°C matches 5°C and 7°C)
	val, ok = lookupBin(bins, 6.0)
	assert.True(t, ok)
	assert.InDelta(t, 4.0, val, 1e-6) // (3.0 + 5.0) / 2

	// Adjacent single +1 match (e.g. 4°C matches bin 5)
	val, ok = lookupBin(bins, 4.0)
	assert.True(t, ok)
	assert.InDelta(t, 3.0, val, 1e-6)

	// No match (e.g. 25°C)
	_, ok = lookupBin(bins, 25.0)
	assert.False(t, ok)

	// Nil bins
	_, ok = lookupBin(nil, 5.0)
	assert.False(t, ok)
}

func TestTileAndTrim(t *testing.T) {
	profile := make([]float64, 96)
	for i := range profile {
		profile[i] = float64(i)
	}

	res := tileAndTrim(profile, 200)
	require.Len(t, res, 200)

	firstSlot := int(time.Now().Truncate(tariff.SlotDuration).Sub(now.BeginningOfDay()) / tariff.SlotDuration)
	assert.Equal(t, float64(firstSlot), res[0], "starts at the current slot")
	assert.Equal(t, res[0], res[96], "wraps after a full day")
}
