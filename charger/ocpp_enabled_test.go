package charger

import (
	"testing"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/stretchr/testify/require"
)

// TestEnabledByStatusPhaseSwitch checks that a SuspendedEVSE within the phase
// switch pause counts as enabled, so chargers that pause for the switch
// (Mennekes AMTRON 4You) do not trigger "charger out of sync".
func TestEnabledByStatusPhaseSwitch(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   core.ChargePointStatus
		switched time.Time
		enabled  bool
		ok       bool
	}{
		{"suspended, no switch", core.ChargePointStatusSuspendedEVSE, time.Time{}, false, true},
		{"suspended, switching", core.ChargePointStatusSuspendedEVSE, time.Now().Add(-time.Minute), true, true},
		{"suspended, pause elapsed", core.ChargePointStatusSuspendedEVSE, time.Now().Add(-phaseSwitchPause), false, true},
		{"charging", core.ChargePointStatusCharging, time.Time{}, true, true},
		{"suspended by ev", core.ChargePointStatusSuspendedEV, time.Time{}, true, true},
		{"preparing undecided", core.ChargePointStatusPreparing, time.Now(), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &OCPP{phasesSwitched: tc.switched}

			enabled, ok := c.enabledByStatus(tc.status)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.enabled, enabled)
		})
	}
}
