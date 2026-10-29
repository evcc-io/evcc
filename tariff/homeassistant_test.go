package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/util/homeassistant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolarForecastRates(t *testing.T) {
	res := map[string]homeassistant.SolarForecast{
		"a": {WhHours: map[string]float64{
			"2026-09-12T13:00:00+00:00": 1000,
			"2026-09-12T12:00:00+00:00": 500,
		}},
		// same hour in a different offset adds up
		"b": {WhHours: map[string]float64{
			"2026-09-12T14:00:00+02:00": 200,
		}},
	}

	data, err := solarForecastRates(res)
	require.NoError(t, err)
	require.Len(t, data, 2)

	start := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	assert.True(t, data[0].Start.Equal(start))
	assert.True(t, data[0].End.Equal(start.Add(time.Hour)))
	assert.Equal(t, 700.0, data[0].Value)
	assert.True(t, data[1].Start.Equal(start.Add(time.Hour)))
	assert.Equal(t, 1000.0, data[1].Value)

	_, err = solarForecastRates(map[string]homeassistant.SolarForecast{
		"a": {WhHours: map[string]float64{"invalid": 1}},
	})
	assert.Error(t, err)
}

func TestSolarForecastRatesHalfHourly(t *testing.T) {
	res := map[string]homeassistant.SolarForecast{
		"solcast": {WhHours: map[string]float64{
			"2026-09-12T07:00:00+00:00": 100,
			"2026-09-12T07:30:00+00:00": 200,
		}},
		"hourly": {WhHours: map[string]float64{
			"2026-09-12T07:00:00+00:00": 50,
		}},
	}

	data, err := solarForecastRates(res)
	require.NoError(t, err)
	require.Len(t, data, 2)

	start := time.Date(2026, 9, 12, 7, 0, 0, 0, time.UTC)
	assert.True(t, data[0].Start.Equal(start))
	assert.True(t, data[0].End.Equal(start.Add(30*time.Minute)))
	assert.Equal(t, 250.0, data[0].Value)
	assert.True(t, data[1].Start.Equal(start.Add(30*time.Minute)))
	assert.True(t, data[1].End.Equal(start.Add(time.Hour)))
	assert.Equal(t, 450.0, data[1].Value)
}

func TestSolarForecastRatesOffGrid(t *testing.T) {
	// Forecast.Solar adds sunrise and sunset keys off the hourly grid
	res := map[string]homeassistant.SolarForecast{
		"forecast_solar": {WhHours: map[string]float64{
			"2026-09-12T06:20:17+02:00": 0,
			"2026-09-12T07:00:00+02:00": 89,
			"2026-09-12T08:00:00+02:00": 376,
			"2026-09-12T08:59:23+02:00": 19,
		}},
	}

	data, err := solarForecastRates(res)
	require.NoError(t, err)
	require.Len(t, data, 2)

	start := time.Date(2026, 9, 12, 5, 0, 0, 0, time.UTC)
	assert.True(t, data[0].Start.Equal(start))
	assert.True(t, data[0].End.Equal(start.Add(time.Hour)))
	assert.Equal(t, 89.0, data[0].Value)
	assert.True(t, data[1].Start.Equal(start.Add(time.Hour)))
	assert.True(t, data[1].End.Equal(start.Add(2*time.Hour)))
	assert.Equal(t, 376.0, data[1].Value)
}

func TestSolarForecastRatesZeroFill(t *testing.T) {
	res := map[string]homeassistant.SolarForecast{
		"a": {WhHours: map[string]float64{
			"2026-09-12T10:00:00+00:00": 100,
			"2026-09-12T13:00:00+00:00": 200,
		}},
	}

	data, err := solarForecastRates(res)
	require.NoError(t, err)
	require.Len(t, data, 4)

	start := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	for i, v := range []float64{100, 0, 0, 200} {
		assert.True(t, data[i].Start.Equal(start.Add(time.Duration(i)*time.Hour)))
		assert.Equal(t, v, data[i].Value)
	}

	data = zeroFillBefore(data, start.Add(-3*time.Hour))
	require.Len(t, data, 7)
	assert.True(t, data[0].Start.Equal(start.Add(-3*time.Hour)))
	assert.Equal(t, 0.0, data[2].Value)
}
