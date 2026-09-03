package core

import (
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	circuitpkg "github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSetLimitDeadlockPrevention(t *testing.T) {
	Voltage = 230
	ctrl := gomock.NewController(t)

	// Create real unmetered circuit with 16A limit
	c, err := circuitpkg.New(util.NewLogger("circuit"), "main", 16, 0, nil, 0)
	require.NoError(t, err)

	charger := api.NewMockCharger(ctrl)

	lp := &Loadpoint{
		log:            util.NewLogger("lp"),
		bus:            evbus.New(),
		clock:          clock.New(),
		charger:        charger,
		circuit:        c,
		wakeUpTimer:    NewTimer(),
		chargeCurrents: nil,
		offeredCurrent: 10,          // We offered 10A last time
		status:         api.StatusB, // Not yet charging
		enabled:        true,
		minCurrent:     6,
		maxCurrent:     16,
		phases:         1,
	}

	siteAPI := &stubSite{
		circuit: c,
		lps:     []loadpoint.API{lp},
	}
	lp.site = siteAPI

	// Initially update the circuit as site does
	err = c.Update([]api.CircuitLoad{lp})
	require.NoError(t, err)

	// Now it should be 0A, because we are not charging.
	assert.Equal(t, 0.0, c.GetMaxPhaseCurrent())

	// We expect charger.MaxCurrent(16) to be called because it shouldn't be capped!
	charger.EXPECT().MaxCurrent(int64(16)).Return(nil)

	// Request 16A. It should succeed (not be capped to 0A)
	err = lp.setLimit(16)
	require.NoError(t, err)
	assert.Equal(t, 16.0, lp.offeredCurrent)
}
