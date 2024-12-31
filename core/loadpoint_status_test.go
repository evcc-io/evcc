package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestStatusEvents(t *testing.T) {
	tc := []struct {
		from, to api.ChargeStatus
		events   []string
	}{
		{api.StatusUnknown, api.StatusDisconnected, []string{evVehicleDisconnect}},
		{api.StatusUnknown, api.StatusConnected, []string{evVehicleConnect}},
		{api.StatusUnknown, api.StatusCharging, []string{evVehicleConnect, evChargeStart}},

		{api.StatusDisconnected, api.StatusConnected, []string{evVehicleConnect}},
		{api.StatusDisconnected, api.StatusCharging, []string{evVehicleConnect, evChargeStart}},

		{api.StatusConnected, api.StatusDisconnected, []string{evVehicleDisconnect}},
		{api.StatusConnected, api.StatusCharging, []string{evChargeStart}},

		{api.StatusCharging, api.StatusDisconnected, []string{evChargeStop, evVehicleDisconnect}},
		{api.StatusCharging, api.StatusConnected, []string{evChargeStop}},
	}

	for _, tc := range tc {
		ev := statusEvents(tc.from, tc.to)
		assert.Equalf(t, tc.events, ev, "from %s to %s got: %v", tc.from, tc.to, ev)
	}
}
