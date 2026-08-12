package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolarMinSocPolicy(t *testing.T) {
	now := time.Date(2026, 8, 10, 10, 30, 0, 0, time.UTC)
	rates := func(energy float64) api.Rates {
		power := energy / solarMinSocHorizon.Hours() * 1e3
		res := make(api.Rates, 0, 74)
		for ts := now.Add(-time.Hour); !ts.After(now.Add(solarMinSocHorizon)); ts = ts.Add(time.Hour) {
			res = append(res, api.Rate{Start: ts, End: ts.Add(time.Hour), Value: power})
		}
		return res
	}

	conf := defaultSolarMinSocConfig()
	conf.Enabled = true
	conf.Vehicles["car"] = api.SolarMinSocVehicleConfig{Low: 60, Medium: 40, High: 10}

	for _, tc := range []struct {
		energy float64
		state  api.SolarMinSocState
		soc    int
	}{
		{4.99, api.SolarMinSocLow, 60},
		{5, api.SolarMinSocMedium, 40},
		{14.99, api.SolarMinSocMedium, 40},
		{15, api.SolarMinSocHigh, 10},
	} {
		var policy solarMinSocPolicy
		require.NoError(t, policy.configure(conf))
		require.True(t, policy.update(rates(tc.energy), now, 1))
		assert.InDelta(t, tc.energy, policy.ForecastEnergy, 1e-9)
		assert.Equal(t, tc.state, policy.State)
		assert.Equal(t, tc.soc, policy.minSoc("car"))
	}
}

func TestSolarMinSocPolicyRetainsLastValidForecast(t *testing.T) {
	now := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	rates := api.Rates{
		{Start: now, End: now.Add(time.Hour), Value: 100},
		{Start: now.Add(solarMinSocHorizon), End: now.Add(solarMinSocHorizon + time.Hour), Value: 100},
	}

	conf := defaultSolarMinSocConfig()
	conf.Enabled = true
	var policy solarMinSocPolicy
	require.NoError(t, policy.configure(conf))
	require.False(t, policy.update(rates, now, 1), "large forecast gaps must be rejected")
	assert.False(t, policy.Available)

	complete := make(api.Rates, 0, 73)
	for ts := now; !ts.After(now.Add(solarMinSocHorizon)); ts = ts.Add(time.Hour) {
		complete = append(complete, api.Rate{Start: ts, End: ts.Add(time.Hour), Value: 100})
	}
	require.True(t, policy.update(complete, now, 1))
	state, energy, updated := policy.State, policy.ForecastEnergy, policy.Updated

	require.False(t, policy.update(complete[:10], now.Add(time.Hour), 1))
	assert.True(t, policy.Available)
	assert.Equal(t, state, policy.State)
	assert.Equal(t, energy, policy.ForecastEnergy)
	assert.Equal(t, updated, policy.Updated)
}

func TestSolarMinSocAvailableVehicles(t *testing.T) {
	site := NewSite()
	site.solarMinSoc.AvailableVehicles = []api.SolarMinSocVehicle{{Name: "stale"}}

	status := site.GetSolarMinSoc()

	assert.NotNil(t, status.AvailableVehicles)
	assert.Empty(t, status.AvailableVehicles)

	status.AvailableVehicles = append(status.AvailableVehicles, api.SolarMinSocVehicle{Name: "caller"})
	assert.Empty(t, site.GetSolarMinSoc().AvailableVehicles)
}

func TestValidateSolarMinSocConfig(t *testing.T) {
	conf := defaultSolarMinSocConfig()
	conf.LowThreshold = 15
	conf.MediumThreshold = 5
	require.Error(t, validateSolarMinSocConfig(conf))

	conf = defaultSolarMinSocConfig()
	conf.Vehicles["car"] = api.SolarMinSocVehicleConfig{Low: 101}
	require.Error(t, validateSolarMinSocConfig(conf))
}
