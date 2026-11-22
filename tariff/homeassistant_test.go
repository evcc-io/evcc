package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/util/homeassistant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fptr returns a pointer to the given Wh value for building test forecasts.
func fptr(v float64) *float64 { return &v }

func TestSolarForecastRates(t *testing.T) {
	res := map[string]homeassistant.SolarForecast{
		"a": {WhHours: map[string]*float64{
			"2026-09-12T13:00:00+00:00": fptr(1000),
			"2026-09-12T12:00:00+00:00": fptr(500),
		}},
		// same hour in a different offset adds up
		"b": {WhHours: map[string]*float64{
			"2026-09-12T14:00:00+02:00": fptr(200),
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
		"a": {WhHours: map[string]*float64{"invalid": fptr(1)}},
	})
	assert.Error(t, err)
}

func TestSolarForecastRatesHalfHourly(t *testing.T) {
	res := map[string]homeassistant.SolarForecast{
		"solcast": {WhHours: map[string]*float64{
			"2026-09-12T07:00:00+00:00": fptr(100),
			"2026-09-12T07:30:00+00:00": fptr(200),
		}},
		"hourly": {WhHours: map[string]*float64{
			"2026-09-12T07:00:00+00:00": fptr(50),
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
		"forecast_solar": {WhHours: map[string]*float64{
			"2026-09-12T06:20:17+02:00": fptr(0),
			"2026-09-12T07:00:00+02:00": fptr(89),
			"2026-09-12T08:00:00+02:00": fptr(376),
			"2026-09-12T08:59:23+02:00": fptr(19),
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
		"a": {WhHours: map[string]*float64{
			"2026-09-12T10:00:00+00:00": fptr(100),
			"2026-09-12T13:00:00+00:00": fptr(200),
		}},
	}

	data, err := solarForecastRates(res)
	require.NoError(t, err)
	require.Len(t, data, 4)

	start := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	for i, v := range []float64{100, 0, 0, 200} {
		assert.True(t, data[i].Start.Equal(start.Add(time.Duration(i)*time.Hour)))
		assert.True(t, data[i].End.Equal(start.Add(time.Duration(i+1)*time.Hour)))
		assert.Equal(t, v, data[i].Value)
	}

	// partial slots at both ends
	data = zeroFill(data, start.Add(-150*time.Minute), start.Add(5*time.Hour+30*time.Minute))
	require.Len(t, data, 9)

	from := start.Add(-150 * time.Minute)
	for i, d := range []time.Duration{30 * time.Minute, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour, 30 * time.Minute} {
		assert.True(t, data[i].Start.Equal(from), "slot %d start", i)
		from = from.Add(d)
		assert.True(t, data[i].End.Equal(from), "slot %d end", i)
	}

	for i, v := range []float64{0, 0, 0, 100, 0, 0, 200, 0, 0} {
		assert.Equal(t, v, data[i].Value, "slot %d value", i)
	}
}

func TestSolarForecastRatesUnavailable(t *testing.T) {
	// null (unavailable) and negative readings must not become usable rates,
	// while a genuine numeric zero stays valid.
	res := map[string]homeassistant.SolarForecast{
		"a": {WhHours: map[string]*float64{
			"2026-09-12T10:00:00+00:00": nil,      // unavailable, dropped
			"2026-09-12T11:00:00+00:00": fptr(0),  // genuine zero, kept
			"2026-09-12T12:00:00+00:00": fptr(100), // kept
			"2026-09-12T13:00:00+00:00": fptr(-50), // negative, dropped
		}},
	}

	data, err := solarForecastRates(res)
	require.NoError(t, err)

	// only the genuine zero and the positive reading remain as usable rates
	require.Len(t, data, 2)

	start := time.Date(2026, 9, 12, 11, 0, 0, 0, time.UTC)
	assert.True(t, data[0].Start.Equal(start))
	assert.Equal(t, 0.0, data[0].Value)
	assert.True(t, data[1].Start.Equal(start.Add(time.Hour)))
	assert.Equal(t, 100.0, data[1].Value)
}
