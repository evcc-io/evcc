package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHeatingSessionStartsFromAZeroCounter: a heater whose counter still stood at 0.0 kWh when the day
// began must book its energy anyway. The session store drops a start meter of zero, which silently
// disabled both the counter-delta booking and the day persistence (measured 2026-10-31: the two new
// electric heaters sat in the sessions table with meter_start_kwh NULL and charged_kwh 0.0).
func TestHeatingSessionStartsFromAZeroCounter(t *testing.T) {
	lp, me, _, _ := heatingLoadpoint(t, []api.Feature{api.Heating, api.IntegratedDevice})
	require.True(t, lp.heatingDevice())

	counter := 0.0
	me.EXPECT().TotalEnergy().DoAndReturn(func() (float64, error) { return counter, nil }).AnyTimes()

	lp.updateDaySession()
	require.NotNil(t, lp.session)
	require.NotNil(t, lp.session.MeterStart, "a counter at 0.0 must still produce a start meter")
	assert.Equal(t, 0.0, *lp.session.MeterStart)
	require.NotNil(t, lp.session.MeterStop, "the day session is visible from the first tick")
	assert.Equal(t, 0.0, *lp.session.MeterStop)

	// the device ran on its own, evcc never switched it on
	counter = 0.018
	lp.updateDaySession()

	require.NotNil(t, lp.session.MeterStop)
	assert.InDelta(t, 0.018, *lp.session.MeterStop, 1e-9)
	assert.InDelta(t, 0.018, lp.session.ChargedEnergy, 1e-9, "the counter delta is booked")
}

// TestHeatingSessionSeedsStartOverAWrittenStop: after a restart the row comes back with a stop meter but
// without a start (it was written that way). The start must be seeded from the meter even then, otherwise
// the rest of the day books nothing.
func TestHeatingSessionSeedsStartOverAWrittenStop(t *testing.T) {
	lp, me, _, clck := heatingLoadpoint(t, []api.Feature{api.Heating, api.IntegratedDevice})

	counter := 0.018
	me.EXPECT().TotalEnergy().DoAndReturn(func() (float64, error) { return counter, nil }).AnyTimes()

	stop := 0.018
	lp.session = &session.Session{Created: clck.Now(), MeterStop: &stop}

	lp.updateDaySession()
	require.NotNil(t, lp.session)
	require.NotNil(t, lp.session.MeterStart, "a restored row without a start still needs one")

	// the next counter step is booked
	counter = 0.030
	lp.updateDaySession()
	assert.InDelta(t, 0.012, lp.session.ChargedEnergy, 1e-9)
}
