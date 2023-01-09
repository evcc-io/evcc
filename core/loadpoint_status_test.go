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
		{api.StatusNone, api.StatusA, []string{evVehicleDisconnect}},
		{api.StatusNone, api.StatusB, []string{evVehicleConnect}},
		{api.StatusNone, api.StatusC, []string{evVehicleConnect, evChargeStart}},

		{api.StatusA, api.StatusB, []string{evVehicleConnect}},
		{api.StatusA, api.StatusC, []string{evVehicleConnect, evChargeStart}},

		{api.StatusB, api.StatusA, []string{evVehicleDisconnect}},
		{api.StatusB, api.StatusC, []string{evChargeStart}},

		{api.StatusC, api.StatusA, []string{evChargeStop, evVehicleDisconnect}},
		{api.StatusC, api.StatusB, []string{evChargeStop}},
	}

	for _, tc := range tc {
		ev := statusEvents(tc.from, tc.to)
		assert.Equalf(t, tc.events, ev, "from %s to %s got: %v", tc.from, tc.to, ev)
	}
}
