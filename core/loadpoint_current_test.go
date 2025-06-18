package core

import (
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestSetPhysicalCurrentRange(t *testing.T) {
	lp := &Loadpoint{
		log:        util.NewLogger("test"),
		clock:      clock.New(),
		bus:        evbus.New(),
		settings:   settings.NewDatabaseSettingsAdapter("foo"),
		minCurrent: 6,
		maxCurrent: 16,
	}

	// Test with valid range
	min, max := 5.0, 32.0
	err := lp.SetPhysicalCurrentRange(&min, &max)
	assert.NoError(t, err)
	assert.Equal(t, min, *lp.getMinPhysicalCurrent())
	assert.Equal(t, max, *lp.getMaxPhysicalCurrent())
	assert.Equal(t, 6.0, lp.minCurrent)  // No adjustment needed
	assert.Equal(t, 16.0, lp.maxCurrent) // No adjustment needed

	// Test with invalid range (min > max)
	min, max = 20.0, 15.0
	err = lp.SetPhysicalCurrentRange(&min, &max)
	assert.Error(t, err)

	// Test with min adjustment needed
	min, max = 10.0, 32.0
	lp.minCurrent = 6
	lp.maxCurrent = 16
	err = lp.SetPhysicalCurrentRange(&min, &max)
	assert.NoError(t, err)
	assert.Equal(t, 10.0, lp.minCurrent) // Should be adjusted to minPhysical

	// Test with max adjustment needed
	min, max = 5.0, 15.0
	lp.minCurrent = 6
	lp.maxCurrent = 16
	err = lp.SetPhysicalCurrentRange(&min, &max)
	assert.NoError(t, err)
	assert.Equal(t, 15.0, lp.maxCurrent) // Should be adjusted to maxPhysical
}

func TestSetMinMaxCurrentWithPhysicalLimits(t *testing.T) {
	lp := &Loadpoint{
		log:        util.NewLogger("test"),
		clock:      clock.New(),
		bus:        evbus.New(),
		settings:   settings.NewDatabaseSettingsAdapter("foo"),
		minCurrent: 6,
		maxCurrent: 16,
	}

	// Set physical limits
	min, max := 5.0, 32.0
	err := lp.SetPhysicalCurrentRange(&min, &max)
	assert.NoError(t, err)

	// Test valid min current
	err = lp.SetMinCurrent(10.0)
	assert.NoError(t, err)
	assert.Equal(t, 10.0, lp.minCurrent)

	// Test min current below physical min
	err = lp.SetMinCurrent(4.0)
	assert.Error(t, err)

	// Test valid max current
	err = lp.SetMaxCurrent(20.0)
	assert.NoError(t, err)
	assert.Equal(t, 20.0, lp.maxCurrent)

	// Test max current above physical max
	err = lp.SetMaxCurrent(40.0)
	assert.Error(t, err)

	// Test min current > max current
	lp.maxCurrent = 15.0
	err = lp.SetMinCurrent(16.0)
	assert.Error(t, err)

	// Test max current < min current
	lp.minCurrent = 10.0
	err = lp.SetMaxCurrent(9.0)
	assert.Error(t, err)
}

func TestPhysicalCurrentRangeAdjustment(t *testing.T) {
	lp := &Loadpoint{
		log:      util.NewLogger("test"),
		clock:    clock.New(),
		bus:      evbus.New(),
		settings: settings.NewDatabaseSettingsAdapter("foo"),
	}

	tests := []struct {
		minPhysical, maxPhysical           float64
		min, max, minExpected, maxExpected float64
	}{
		{5, 32, 10, 16, 10, 16}, // No adjustment needed
		{10, 32, 6, 16, 10, 16}, // Min current below physical min
		{5, 16, 6, 32, 6, 16},   // Max current above physical max
		{10, 20, 6, 32, 10, 20}, // Both currents outside physical range
		{5, 16, 20, 32, 16, 16}, // Min current > max physical current
		{10, 32, 6, 8, 10, 10},  // Max current < min physical current
	}

	for _, tc := range tests {
		lp.minCurrent = tc.min
		lp.maxCurrent = tc.max

		minPhysical, maxPhysical := tc.minPhysical, tc.maxPhysical

		err := lp.SetPhysicalCurrentRange(&minPhysical, &maxPhysical)

		assert.NoError(t, err)
		assert.Equal(t, tc.minExpected, lp.minCurrent)
		assert.Equal(t, tc.maxExpected, lp.maxCurrent)
	}
}
