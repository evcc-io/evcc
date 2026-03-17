package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCompute24hAverageTemperature(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tc := []struct {
		name     string
		rates    []api.Rate
		expected float64
		ok       bool
	}{
		{
			name: "happy path - 24h of data",
			rates: []api.Rate{
				{Start: now.Add(-23 * time.Hour), Value: 5.0},
				{Start: now.Add(-12 * time.Hour), Value: 10.0},
				{Start: now.Add(-1 * time.Hour), Value: 15.0},
			},
			expected: 10.0, // (5 + 10 + 15) / 3
			ok:       true,
		},
		{
			name:     "empty rates",
			rates:    []api.Rate{},
			expected: 0,
			ok:       false,
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			avg, ok := compute24hAverageTemperature(tc.rates, now)
			assert.Equal(t, tc.ok, ok)
			if ok {
				assert.InDelta(t, tc.expected, avg, 0.01)
			}
		})
	}
}

func TestApplyTemperatureCorrection_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now().Truncate(15 * time.Minute)

	mockTariff := api.NewMockTariff(ctrl)

	rates := []api.Rate{}

	// Past 7 days - create slots for ALL hours to ensure historical data exists
	for day := -7; day < 0; day++ {
		for hour := 0; hour < 24; hour++ {
			for slot := 0; slot < 4; slot++ {
				baseTime := now.AddDate(0, 0, day).Truncate(24 * time.Hour)
				rates = append(rates, api.Rate{
					Start: baseTime.Add(time.Duration(hour)*time.Hour + time.Duration(slot)*15*time.Minute),
					End:   baseTime.Add(time.Duration(hour)*time.Hour + time.Duration(slot+1)*15*time.Minute),
					Value: 10.0,
				})
			}
		}
	}

	// Past 24h: 5°C (below threshold, heating active)
	for i := -96; i < 0; i++ {
		rates = append(rates, api.Rate{
			Start: now.Add(time.Duration(i) * 15 * time.Minute),
			End:   now.Add(time.Duration(i+1) * 15 * time.Minute),
			Value: 5.0,
		})
	}

	// Future forecast: 8 slots (2 hours)
	// First hour: 5°C, Second hour: 15°C
	for i := 0; i < 8; i++ {
		temp := 5.0
		if i >= 4 {
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
		log:                util.NewLogger("test"),
		HeatingThreshold:   15.0,
		HeatingCoefficient: 0.05,
		tariffs:            &tariff.Tariffs{Temperature: mockTariff},
	}

	profile := []float64{2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0}

	result := site.applyTemperatureCorrection(profile)

	require.Len(t, result, 8)

	// Verify correction is applied: first hour should increase, second hour should decrease
	assert.Greater(t, result[0], 2.0, "first hour should increase (colder forecast)")
	assert.Less(t, result[4], 2.0, "second hour should decrease (warmer forecast)")
	assert.Greater(t, result[0], result[4], "first hour should be higher than second hour")
}

func TestApplyTemperatureCorrection_HeatingInactive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now().Truncate(15 * time.Minute)

	mockTariff := api.NewMockTariff(ctrl)

	// Past 24h average: 20°C (above threshold of 15°C, so heating is inactive)
	rates := []api.Rate{}
	for i := -96; i < 0; i++ {
		rates = append(rates, api.Rate{
			Start: now.Add(time.Duration(i) * 15 * time.Minute),
			End:   now.Add(time.Duration(i+1) * 15 * time.Minute),
			Value: 20.0,
		})
	}

	mockTariff.EXPECT().Rates().Return(rates, nil).AnyTimes()

	site := &Site{
		log:                util.NewLogger("test"),
		HeatingThreshold:   15.0,
		HeatingCoefficient: 0.05,
		tariffs:            &tariff.Tariffs{Temperature: mockTariff},
	}

	profile := []float64{1.0, 2.0, 3.0}
	result := site.applyTemperatureCorrection(profile)

	// Should return unchanged profile when heating is inactive
	assert.Equal(t, profile, result)
}

func TestSumProfiles(t *testing.T) {
	tc := []struct {
		name     string
		profiles [][]float64
		expected []float64
	}{
		{
			name: "two profiles same length",
			profiles: [][]float64{
				{1.0, 2.0, 3.0},
				{4.0, 5.0, 6.0},
			},
			expected: []float64{5.0, 7.0, 9.0},
		},
		{
			name:     "empty profiles",
			profiles: [][]float64{},
			expected: nil,
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			result := sumProfiles(tc.profiles)
			assert.Equal(t, tc.expected, result)
		})
	}
}
