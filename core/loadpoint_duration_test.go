package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// duration-based plan reports the remaining run time as required duration
func TestPlanDurationRequiredDuration(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.planDuration = 30 * time.Minute

	// nothing run yet
	assert.Equal(t, 30*time.Minute, lp.getPlanRequiredDuration(0, 1e4))

	// 10min already run since plan set
	lp.chargeDuration = 10 * time.Minute
	assert.Equal(t, 20*time.Minute, lp.getPlanRequiredDuration(0, 1e4))

	// offset (plan set mid-session) is subtracted
	lp.planDurationOffset = 5 * time.Minute
	assert.Equal(t, 25*time.Minute, lp.getPlanRequiredDuration(0, 1e4))

	// goal met - never negative
	lp.chargeDuration = 40 * time.Minute
	assert.Equal(t, time.Duration(0), lp.getPlanRequiredDuration(0, 1e4))
}

// energy and duration plans are mutually exclusive
func TestPlanDurationEnergyExclusive(t *testing.T) {
	clk := clock.NewMock()
	lp := &Loadpoint{
		log:      util.NewLogger("foo"),
		clock:    clk,
		settings: settings.NewDatabaseSettingsAdapter("test-plan-duration"),
	}
	uiChan, pushChan, lpChan := createChannels(t)
	attachChannels(lp, uiChan, pushChan, lpChan)

	future := clk.Now().Add(2 * time.Hour)

	// energy plan
	require.NoError(t, lp.SetPlanEnergy(future, 5))
	_, energy := lp.getPlanEnergy()
	assert.Equal(t, 5.0, energy)

	// duration plan clears energy, keeps time
	require.NoError(t, lp.SetPlanDuration(future, 30*time.Minute))
	_, energy = lp.getPlanEnergy()
	assert.Equal(t, 0.0, energy, "energy cleared")
	_, dur := lp.getPlanDuration()
	assert.Equal(t, 30*time.Minute, dur)
	assert.Equal(t, future, lp.planTime)

	// energy plan clears duration
	require.NoError(t, lp.SetPlanEnergy(future, 7))
	_, dur = lp.getPlanDuration()
	assert.Equal(t, time.Duration(0), dur, "duration cleared")

	// past target rejected
	assert.Error(t, lp.SetPlanDuration(clk.Now().Add(-time.Hour), 30*time.Minute))
}
