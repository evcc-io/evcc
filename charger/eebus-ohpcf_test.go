package charger

import (
	"testing"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cem/ohpcf"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// methods accessing the compressor entity must error when it is absent,
// so a missing compressor is not mistaken for an idle device.
func TestEEBusOHPCFNotConnected(t *testing.T) {
	c := &EEBusOHPCF{}

	status, err := c.Status()
	require.ErrorIs(t, err, errNotConnected)
	assert.Equal(t, api.StatusNone, status)

	_, err = c.Enabled()
	require.ErrorIs(t, err, errNotConnected)

	require.ErrorIs(t, c.Enable(true), errNotConnected)
	require.ErrorIs(t, c.MaxCurrent(16), errNotConnected)

	// dimming uses EG LPC, which is unavailable without an LPC entity
	require.ErrorIs(t, c.Dim(true), api.ErrNotAvailable)
}

// status mapping: running is C, every other connected state (incl. completed
// and stopped after a boost) is standby B, never disconnected.
func TestOHPCFStatus(t *testing.T) {
	tc := []struct {
		state ucapi.CompressorPowerConsumptionStateType
		want  api.ChargeStatus
	}{
		{ucapi.CompressorPowerConsumptionStateRunning, api.StatusC},
		{ucapi.CompressorPowerConsumptionStateAvailable, api.StatusB},
		{ucapi.CompressorPowerConsumptionStateScheduled, api.StatusB},
		{ucapi.CompressorPowerConsumptionStatePaused, api.StatusB},
		{ucapi.CompressorPowerConsumptionStateCompleted, api.StatusB},
		{ucapi.CompressorPowerConsumptionStateStopped, api.StatusB},
	}

	for _, tc := range tc {
		assert.Equal(t, tc.want, ohpcfStatus(tc.state), "%v", tc.state)
	}
}

// on/off control: enable schedules/resumes, disable stops, and an already
// running/scheduled process issues no further command.
func TestOHPCFControlAction(t *testing.T) {
	tc := []struct {
		state  ucapi.CompressorPowerConsumptionStateType
		enable bool
		want   ohpcfAction
	}{
		{ucapi.CompressorPowerConsumptionStateAvailable, true, ohpcfSchedule},
		{ucapi.CompressorPowerConsumptionStatePaused, true, ohpcfResume},
		{ucapi.CompressorPowerConsumptionStateScheduled, true, ohpcfNone},
		{ucapi.CompressorPowerConsumptionStateRunning, true, ohpcfNone},
		{ucapi.CompressorPowerConsumptionStateRunning, false, ohpcfStop},
		{ucapi.CompressorPowerConsumptionStateScheduled, false, ohpcfStop},
		{ucapi.CompressorPowerConsumptionStateAvailable, false, ohpcfNone},
		{ucapi.CompressorPowerConsumptionStatePaused, false, ohpcfNone},
	}

	for _, tc := range tc {
		assert.Equal(t, tc.want, ohpcfControlAction(tc.state, tc.enable), "%v enable=%v", tc.state, tc.enable)
	}
}

// a consumption-state update always records the compressor entity; while
// disabled it must not attempt to apply (avoids acting on a stale intent, #31549).
func TestOHPCFUseCaseEventConsumptionStateDisabled(t *testing.T) {
	c := &EEBusOHPCF{}
	entity := spinemocks.NewEntityRemoteInterface(t)

	c.UseCaseEvent(nil, entity, ohpcf.DataUpdateConsumptionState)

	got, ok := c.connectedCompressor()
	require.True(t, ok)
	assert.Equal(t, entity, got)
}
