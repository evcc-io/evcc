package core

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestBatteryCoversStart(t *testing.T) {
	Voltage = 230 // 3p at 6A needs 4140W

	for _, tc := range []struct {
		name             string
		sitePower        float64
		effectiveCurrent float64
		limit            *float64
		batteryMode      api.BatteryMode
		expected         bool
	}{
		{"no limit configured", -2000, 0, nil, api.BatteryNormal, true},
		{"empty battery", -4000, 0, new(0.0), api.BatteryNormal, false},
		{"site deficit adds to the gap", 500, 0, new(4200.0), api.BatteryNormal, false},
		{"existing draw reduces the gap", 0, 5, new(800.0), api.BatteryNormal, true},
		{"uncontrolled battery discharges", -4000, 0, new(800.0), api.BatteryUnknown, true},
		{"held battery does not discharge", -4000, 0, new(800.0), api.BatteryHold, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lp := &Loadpoint{
				log:  util.NewLogger("foo"),
				site: &mockSite{maxDischargePower: tc.limit, batteryMode: tc.batteryMode},
			}

			assert.Equal(t, tc.expected, lp.batteryCoversStart(tc.sitePower, minA, tc.effectiveCurrent, 3))
		})
	}
}

// newAssistLoadpoint returns a loadpoint where 3p at 6A needs 1800W
func newAssistLoadpoint(limit float64) *Loadpoint {
	Voltage = 100

	return &Loadpoint{
		log:            util.NewLogger("foo"),
		clock:          clock.NewMock(),
		site:           &mockSite{maxDischargePower: &limit},
		minCurrent:     minA,
		maxCurrent:     maxA,
		phases:         3,
		measuredPhases: 3,
		status:         api.StatusB,
	}
}

// a refused start must reach the caller as zero, not fall through to min current
func TestPvBatteryStartLimited(t *testing.T) {
	lp := newAssistLoadpoint(800)

	// 600W surplus leaves 1200W missing
	assert.Equal(t, 0.0, lp.pvMaxCurrent(api.ModePV, -600, 0, false, true))
}

func TestPvBatteryStartBoost(t *testing.T) {
	// an active boost owns the battery budget and is not second-guessed
	t.Run("boost bypasses the check", func(t *testing.T) {
		lp := newAssistLoadpoint(300)
		lp.batteryBoost = boostStart

		assert.Equal(t, minA, lp.pvMaxCurrent(api.ModePV, -600, 0, false, true))
	})

	// a held boost does not drain the battery, so the check still applies
	t.Run("hold is checked", func(t *testing.T) {
		lp := newAssistLoadpoint(800)
		lp.batteryBoost = boostHold

		assert.Equal(t, 0.0, lp.pvMaxCurrent(api.ModePV, -600, 0, false, true))
	})
}

// the check gates the start only, an enabled loadpoint is left to pv hysteresis
func TestPvBatteryStartEnabled(t *testing.T) {
	lp := newAssistLoadpoint(500)
	lp.status = api.StatusC
	lp.enabled = true
	lp.offeredCurrent = minA
	lp.Disable = loadpoint.ThresholdConfig{}

	assert.Equal(t, minA, lp.pvMaxCurrent(api.ModePV, 600, 0, false, true))
}
