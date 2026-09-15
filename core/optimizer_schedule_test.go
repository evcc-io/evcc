package core

import (
	"slices"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	optimizer "github.com/evcc-io/optimizer/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptimizerSchedule(t *testing.T) {
	boundary := time.Date(2025, 1, 1, 12, 15, 0, 0, time.UTC)
	schedule := optimizerSchedule{
		timestamps: []time.Time{boundary.Add(-2 * time.Second), boundary, boundary.Add(15 * time.Minute)},
		dt:         []int{2, 900, 900},
	}

	for _, tc := range []struct {
		name      string
		at        time.Time
		want      int
		remaining []int
	}{
		{"partial slot", boundary.Add(-time.Second), 0, []int{0, 1, 2}},
		{"last instant", boundary.Add(-time.Nanosecond), 0, []int{0, 1, 2}},
		{"next slot at boundary", boundary, 1, []int{1, 2}},
		{"delayed response", boundary.Add(4 * time.Second), 1, []int{1, 2}},
		{"following slot", boundary.Add(15 * time.Minute), 2, []int{2}},
		{"expired result", boundary.Add(30 * time.Minute), -1, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, schedule.activeSlot(tc.at))
			assert.Equal(t, tc.remaining, slices.Collect(schedule.endsAfter(tc.at)))
		})
	}

	assert.Equal(t, boundary, schedule.end(0))
	assert.Equal(t, boundary.Add(15*time.Minute), schedule.end(1))
	assert.Equal(t, 2*time.Second, schedule.duration(0))
	assert.Equal(t, 15*time.Minute, schedule.duration(1))
	for _, slot := range []int{-1, 3} {
		assert.Zero(t, schedule.end(slot))
		assert.Zero(t, schedule.duration(slot))
	}

	for _, empty := range []optimizerSchedule{
		{},
		{timestamps: []time.Time{boundary}},
		{dt: []int{900}},
	} {
		assert.Equal(t, -1, empty.activeSlot(boundary))
		assert.Empty(t, slices.Collect(empty.endsAfter(boundary)))
	}
}

func TestApplyOptimizerResultSchedule(t *testing.T) {
	boundary := time.Date(2025, 1, 1, 12, 15, 0, 0, time.UTC)
	schedule := optimizerSchedule{
		timestamps: []time.Time{boundary.Add(-2 * time.Second), boundary, boundary.Add(15 * time.Minute)},
		dt:         []int{2, 900, 900},
	}
	req := optimizer.OptimizationInput{
		TimeSeries: optimizer.TimeSeries{Dt: schedule.dt},
		Batteries:  []optimizer.BatteryConfig{{SCapacity: 1000, SInitial: 500, SMax: 1000}},
	}
	details := requestDetails{
		Timestamps:     schedule.timestamps,
		BatteryDetails: []batteryDetail{{Type: batteryTypeBattery, Name: "home", controllable: true}},
	}
	res := optimizer.OptimizationResult{
		GridImport: []float32{1, 250, 0},
		Batteries: []optimizer.BatteryResult{{
			ChargingPower:    []float32{1, 250, 0},
			DischargingPower: []float32{0, 0, 0},
			StateOfCharge:    []float32{1000, 500, 800},
		}},
	}

	for _, tc := range []struct {
		name    string
		at      time.Time
		power   float64
		full    time.Time
		highest time.Time
	}{
		{"partial slot", boundary.Add(-time.Second), 1800, boundary, boundary},
		{"delayed response", boundary.Add(4 * time.Second), 1000, time.Time{}, boundary.Add(30 * time.Minute)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := make(chan util.Param, 10)
			site := &Site{valueChan: values}
			site.applyOptimizerResult(req, details, res, schedule, tc.at, tc.at)

			s, ok := site.suggestions[batteryKey("home")]
			require.True(t, ok)
			assert.Equal(t, api.BatteryCharge.String(), s.Action)
			assert.InDelta(t, tc.power, s.Charge, 1e-3)
			assert.InDelta(t, tc.power, s.Grid, 1e-3)
			require.NotNil(t, site.battery.Forecast)
			require.NotNil(t, site.battery.Forecast.Highest)
			assert.Equal(t, tc.highest, site.battery.Forecast.Highest.Time)

			close(values)
			var batteries []batteryResult
			for val := range values {
				if val.Key == "evopt-batteries" {
					batteries = val.Val.([]batteryResult)
				}
			}
			require.Len(t, batteries, 1)
			assert.Equal(t, tc.full, batteries[0].Full)
		})
	}
}

func TestLoadpointPlan(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	schedule := optimizerSchedule{
		timestamps: []time.Time{now.Add(-15 * time.Minute), now, now.Add(15 * time.Minute), now.Add(30 * time.Minute)},
		dt:         []int{900, 900, 900, 900},
	}

	// expired slot, charging, noise, charging
	res := optimizer.BatteryResult{ChargingPower: []float32{1000, 500, 5, 250}}
	prices := []float32{0.1e-3, 0.2e-3, 0.3e-3, 0.4e-3}

	plan := loadpointPlan(res, prices, schedule, now)

	require.Len(t, plan.rates, 2)
	assert.Equal(t, now, plan.rates[0].Start)
	assert.Equal(t, now.Add(15*time.Minute), plan.rates[0].End)
	assert.InDelta(t, 0.2, plan.rates[0].Value, 1e-6)
	assert.Equal(t, now.Add(30*time.Minute), plan.rates[1].Start)
	assert.InDelta(t, 0.4, plan.rates[1].Value, 1e-6)
	assert.InDelta(t, 750, plan.energy, 1e-6)
}
