package cupra

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderStatus covers the Cupra v5 charging-status payload variants
// captured in issues #30045 and #30118.
func TestProviderStatus(t *testing.T) {
	cases := []struct {
		name              string
		statusField       string
		batteryCardStatus string
		want              api.ChargeStatus
	}{
		// cable not plugged in: API emits batteryCardStatus=notConnected
		{"unplugged", "NotReadyForCharging", "notConnected", api.StatusA},
		// charging: API omits batteryCardStatus
		{"charging", "Charging", "", api.StatusC},
		// plugged but not charging (e.g. chargeMode=off): API omits
		// batteryCardStatus, Status reports NotReadyForCharging — must NOT be
		// misread as disconnected (#30118)
		{"plugged_paused", "NotReadyForCharging", "", api.StatusB},
		// plugged, ready to charge but not yet started
		{"ready", "ReadyForCharging", "", api.StatusB},
		// already-connected pre-v5 payload
		{"connected", "Connected", "", api.StatusB},
		// fault / error while plugged
		{"error", "error", "", api.StatusB},
		// target SoC reached in myCupra, cable still connected (#30118)
		{"soc_reached", "ChargePurposeReachedAndNotConservationCharging", "", api.StatusB},
		// unknown future status with no card info defaults to disconnected
		{"unknown", "SomethingNew", "", api.StatusA},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var s Status
			s.Services.Charging.Status = tc.statusField
			s.Services.Charging.BatteryCardStatus = tc.batteryCardStatus

			v := &Provider{
				statusG: func() (Status, error) { return s, nil },
			}

			got, err := v.Status()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
