package charger

import (
	"testing"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestOHPCFStatus(t *testing.T) {
	cases := []struct {
		state   ucapi.CompressorPowerConsumptionStateType
		status  api.ChargeStatus
		enabled bool
	}{
		{ucapi.CompressorPowerConsumptionStateAvailable, api.StatusB, false},
		{ucapi.CompressorPowerConsumptionStateScheduled, api.StatusB, true},
		{ucapi.CompressorPowerConsumptionStateRunning, api.StatusC, true},
		{ucapi.CompressorPowerConsumptionStatePaused, api.StatusB, false},
		{ucapi.CompressorPowerConsumptionStateCompleted, api.StatusA, false},
		{ucapi.CompressorPowerConsumptionStateStopped, api.StatusA, false},
		{ucapi.CompressorPowerConsumptionStateType("unknown"), api.StatusA, false},
	}

	for _, tc := range cases {
		t.Run(string(tc.state), func(t *testing.T) {
			assert.Equal(t, tc.status, ohpcfStatus(tc.state), "status")
			assert.Equal(t, tc.enabled, ohpcfEnabled(tc.state), "enabled")
		})
	}
}

func TestOHPCFControlAction(t *testing.T) {
	cases := []struct {
		state      ucapi.CompressorPowerConsumptionStateType
		sufficient bool
		want       ohpcfAction
	}{
		// enough surplus: start when available, resume when paused, otherwise hold
		{ucapi.CompressorPowerConsumptionStateAvailable, true, ohpcfSchedule},
		{ucapi.CompressorPowerConsumptionStatePaused, true, ohpcfResume},
		{ucapi.CompressorPowerConsumptionStateScheduled, true, ohpcfNone},
		{ucapi.CompressorPowerConsumptionStateRunning, true, ohpcfNone},
		// not enough surplus: stop while running or scheduled, otherwise hold
		{ucapi.CompressorPowerConsumptionStateRunning, false, ohpcfStop},
		{ucapi.CompressorPowerConsumptionStateScheduled, false, ohpcfStop},
		{ucapi.CompressorPowerConsumptionStatePaused, false, ohpcfNone},
		{ucapi.CompressorPowerConsumptionStateAvailable, false, ohpcfNone},
	}

	for _, tc := range cases {
		t.Run(string(tc.state), func(t *testing.T) {
			assert.Equal(t, tc.want, ohpcfControlAction(tc.state, tc.sufficient))
		})
	}
}
