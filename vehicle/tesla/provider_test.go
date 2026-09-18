package tesla

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/tesla-proxy-client"
	"github.com/stretchr/testify/require"
)

func TestProviderStatus(t *testing.T) {
	for _, tc := range []struct {
		state  string
		status api.ChargeStatus
		err    bool
	}{
		{"Disconnected", api.StatusA, false},
		{"Stopped", api.StatusB, false},
		{"NoPower", api.StatusB, false},
		{"Complete", api.StatusB, false},
		{"Starting", api.StatusB, false},
		{"Charging", api.StatusC, false},
		{"", api.StatusNone, true},
		{"Calibrating", api.StatusNone, true},
	} {
		t.Run(tc.state, func(t *testing.T) {
			p := &Provider{dataG: func() (*tesla.VehicleData, error) {
				var res tesla.VehicleData
				res.Response.ChargeState.ChargingState = tc.state
				return &res, nil
			}}

			status, err := p.Status()
			require.Equal(t, tc.err, err != nil, err)
			require.Equal(t, tc.status, status)
		})
	}
}
