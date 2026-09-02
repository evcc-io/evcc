package devicehost

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMethodName guards the wire names derived from method expressions against
// a change in the Go runtime's function naming
func TestMethodName(t *testing.T) {
	for _, tc := range []struct {
		expr       any
		capability string
		method     string
	}{
		{api.Meter.CurrentPower, "api.Meter", "CurrentPower"},
		{api.ChargeState.Status, "api.ChargeState", "Status"},
		{api.Charger.Enable, "api.Charger", "Enable"},
		{api.CurrentController.MaxCurrent, "api.CurrentController", "MaxCurrent"},
		{api.PhaseCurrents.Currents, "api.PhaseCurrents", "Currents"},
	} {
		capability, method, err := methodName(tc.expr)
		require.NoError(t, err)
		assert.Equal(t, tc.capability, capability)
		assert.Equal(t, tc.method, method)
	}
}

func TestMethodNameInvalid(t *testing.T) {
	_, _, err := methodName(42)
	assert.Error(t, err)
}

// TestCapabilityKeys ensures the generated table is keyed by the wire names
func TestCapabilityKeys(t *testing.T) {
	require.Contains(t, capTable, "api.MeterEnergy")
	require.Contains(t, capTable, "api.PhaseSwitcher")
}
