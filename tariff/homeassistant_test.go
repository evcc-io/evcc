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
