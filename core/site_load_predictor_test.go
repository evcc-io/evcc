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
	// First hour: 5°C, second hour: 15°C, third hour: 20°C (above heating stop threshold)
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

	result := site.applyTemperatureCorrection(profile)

	require.Len(t, result, 12)

	// Verify correction is applied: first hour should increase, second hour should decrease
	assert.Greater(t, result[0], 2.0, "first hour should increase (colder forecast)")
	assert.Less(t, result[4], 2.0, "second hour should decrease (warmer forecast)")
	assert.Greater(t, result[0], result[4], "first hour should be higher than second hour")
	// above the heating threshold the correction is skipped but the historical average is kept
	assert.Equal(t, 2.0, result[8], "third hour keeps historical average when above heating threshold")
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
