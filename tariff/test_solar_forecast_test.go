package tariff

import (
	"context"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolarForecastStatic(t *testing.T) {
	// Test static solar forecast template
	tariff, err := NewFromConfig(context.TODO(), "template", map[string]any{
		"template": "test-solar-forecast-static",
		"power":    3000.0,
		"interval": "1h",
	})
	require.NoError(t, err)

	rates, err := tariff.Rates()
	require.NoError(t, err)

	// Should have 72 hours of data (3 days)
	assert.Equal(t, 72, len(rates))

	// All rates should have the same power value
	for _, rate := range rates {
		assert.Equal(t, 3000.0, rate.Value, "All rates should have constant power value")
		assert.Equal(t, time.Hour, rate.End.Sub(rate.Start), "Each rate should be 1 hour duration")
	}

	// Check tariff type
	assert.Equal(t, api.TariffTypeSolar, tariff.Type())
}

func TestSolarForecastCurve(t *testing.T) {
	// Test bell curve solar forecast template
	tariff, err := NewFromConfig(context.TODO(), "template", map[string]any{
		"template": "test-solar-forecast-curve",
		"peak":     4500.0,
		"sunrise":  6,
		"sunset":   18,
		"interval": "1h",
	})
	require.NoError(t, err)

	rates, err := tariff.Rates()
	require.NoError(t, err)

	// Should have 72 hours of data (3 days)
	assert.Equal(t, 72, len(rates))

	// Check that we have all three days with similar patterns
	day1Rates := rates[:24]
	day2Rates := rates[24:48]
	day3Rates := rates[48:72]

	// Verify bell curve characteristics for day 1
	validateBellCurve(t, day1Rates, 4500.0, 6, 18)

	// Verify bell curve characteristics for day 2 (should be similar)
	validateBellCurve(t, day2Rates, 4500.0, 6, 18)

	// Verify bell curve characteristics for day 3 (should be similar)
	validateBellCurve(t, day3Rates, 4500.0, 6, 18)

	// Check tariff type
	assert.Equal(t, api.TariffTypeSolar, tariff.Type())
}

func validateBellCurve(t *testing.T, rates []api.Rate, expectedPeak float64, sunrise, sunset int) {
	t.Helper()

	// Find the maximum value (should be around noon)
	var maxValue float64
	var maxHour int

	for i, rate := range rates {
		if rate.Value > maxValue {
			maxValue = rate.Value
			maxHour = i
		}

		// Check hourly duration
		assert.Equal(t, time.Hour, rate.End.Sub(rate.Start), "Each rate should be 1 hour duration")
	}

	// Peak should be close to the expected value (within 10% tolerance for bell curve calculation)
	assert.InDelta(t, expectedPeak, maxValue, expectedPeak*0.1, "Peak value should be close to expected")

	// Peak should occur around noon (sunrise + (sunset-sunrise)/2)
	expectedNoon := sunrise + (sunset-sunrise)/2
	assert.InDelta(t, expectedNoon, maxHour, 1, "Peak should occur around noon")

	// Values before sunrise should be 0
	for i := 0; i < sunrise; i++ {
		assert.Equal(t, 0.0, rates[i].Value, "Power before sunrise should be 0")
	}

	// Values after sunset should be 0
	for i := sunset + 1; i < 24; i++ {
		assert.Equal(t, 0.0, rates[i].Value, "Power after sunset should be 0")
	}

	// Values should increase from sunrise to noon
	for i := sunrise; i < maxHour; i++ {
		assert.LessOrEqual(t, rates[i].Value, rates[i+1].Value, "Power should increase towards noon")
	}

	// Values should decrease from noon to sunset
	for i := maxHour; i < sunset; i++ {
		assert.GreaterOrEqual(t, rates[i].Value, rates[i+1].Value, "Power should decrease after noon")
	}
}

func TestSolarForecastTemplateDefaults(t *testing.T) {
	// Test that templates work with default values
	tests := []struct {
		name     string
		template string
		checks   func(t *testing.T, rates api.Rates)
	}{
		{
			name:     "static default",
			template: "test-solar-forecast-static",
			checks: func(t *testing.T, rates api.Rates) {
				// Default power should be 2000W
				for _, rate := range rates {
					assert.Equal(t, 2000.0, rate.Value)
				}
			},
		},
		{
			name:     "curve default",
			template: "test-solar-forecast-curve",
			checks: func(t *testing.T, rates api.Rates) {
				// Should have bell curve with default parameters
				day1Rates := rates[:24]
				validateBellCurve(t, day1Rates, 4500.0, 6, 18)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tariff, err := NewFromConfig(context.TODO(), "template", map[string]any{
				"template": tt.template,
			})
			require.NoError(t, err)

			rates, err := tariff.Rates()
			require.NoError(t, err)

			assert.Equal(t, 72, len(rates), "Should have 72 hours of data")
			tt.checks(t, rates)
		})
	}
}

func TestSolarTemplatesCaching(t *testing.T) {
	// Test that template-based tariffs properly use caching
	config := map[string]any{
		"template": "test-solar-forecast-static",
		"power":    5000.0,
		"interval": "1h",
	}

	// Create first tariff instance
	tariff1, err := NewFromConfig(context.TODO(), "template", config)
	require.NoError(t, err)

	// Get rates (should cache the result)
	rates1, err := tariff1.Rates()
	require.NoError(t, err)
	require.NotEmpty(t, rates1)

	// Create second tariff instance with same config
	tariff2, err := NewFromConfig(context.TODO(), "template", config)
	require.NoError(t, err)

	// Get rates (should use cached result)
	rates2, err := tariff2.Rates()
	require.NoError(t, err)

	// Verify rates are identical (indicating cache was used)
	assert.Equal(t, len(rates1), len(rates2), "Cached rates should have same length")
	for i, rate1 := range rates1 {
		assert.Equal(t, rate1.Start, rates2[i].Start, "Cached start time should match for rate %d", i)
		assert.Equal(t, rate1.End, rates2[i].End, "Cached end time should match for rate %d", i)
		assert.Equal(t, rate1.Value, rates2[i].Value, "Cached value should match for rate %d", i)
	}

	// Test different config creates different cache
	differentConfig := map[string]any{
		"template": "test-solar-forecast-static",
		"power":    3000.0, // Different power value
		"interval": "1h",
	}

	tariff3, err := NewFromConfig(context.TODO(), "template", differentConfig)
	require.NoError(t, err)

	rates3, err := tariff3.Rates()
	require.NoError(t, err)

	// Verify different config produces different rates
	assert.Equal(t, len(rates1), len(rates3), "Different config should still have same length")
	for i, rate1 := range rates1 {
		assert.NotEqual(t, rate1.Value, rates3[i].Value, "Different config should produce different values for rate %d", i)
		assert.Equal(t, 3000.0, rates3[i].Value, "New config should use specified power value for rate %d", i)
	}
}
