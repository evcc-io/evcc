package charger

import (
	"testing"

	eebusapi "github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cem/ohpcf"
	"github.com/enbility/eebus-go/usecases/ma/mpc"
	ucmocks "github.com/enbility/eebus-go/usecases/mocks"
	spineapi "github.com/enbility/spine-go/api"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
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
	require.ErrorIs(t, c.dim(true), api.ErrNotAvailable)
}

// a failed enable must not persist the intent, otherwise Enabled() reports a
// state the compressor never accepted and the loadpoint runs out of sync (#32252).
func TestOHPCFEnableFailureKeepsState(t *testing.T) {
	c := &EEBusOHPCF{}

	require.ErrorIs(t, c.Enable(true), errNotConnected)
	assert.False(t, c.lastEnabled())
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
	c := &EEBusOHPCF{connector: eebus.NewConnector()}
	entity := spinemocks.NewEntityRemoteInterface(t)

	c.UseCaseEvent(nil, entity, ohpcf.DataUpdateConsumptionState)

	got, ok := c.connectedCompressor()
	require.True(t, ok)
	assert.Equal(t, entity, got)
}

// a device may remove and re-add its entities while connected (#36631): the removal
// drops the cached entity, so the re-added instance replaces the stale one.
func TestOHPCFEntityReAdded(t *testing.T) {
	uc := ucmocks.NewMaMPCInterface(t)
	c := &EEBusOHPCF{ma: &eebus.MonitoringAppliance{MaMPCInterface: uc}}

	stale := spinemocks.NewEntityRemoteInterface(t)
	fresh := spinemocks.NewEntityRemoteInterface(t)
	sibling := spinemocks.NewEntityRemoteInterface(t)

	listed := func(entities ...spineapi.EntityRemoteInterface) {
		var res []eebusapi.RemoteEntityScenarios
		for _, e := range entities {
			res = append(res, eebusapi.RemoteEntityScenarios{Entity: e})
		}
		uc.EXPECT().RemoteEntitiesScenarios().Return(res).Once()
	}

	listed(stale)
	c.UseCaseEvent(nil, stale, mpc.UseCaseSupportUpdate)
	assert.Same(t, stale, c.mpcEntity)

	// removal of another entity keeps the cached one
	listed(stale)
	c.UseCaseEvent(nil, sibling, mpc.UseCaseSupportUpdate)
	assert.Same(t, stale, c.mpcEntity)

	listed()
	c.UseCaseEvent(nil, stale, mpc.UseCaseSupportUpdate)
	assert.Nil(t, c.mpcEntity)

	listed(fresh)
	c.UseCaseEvent(nil, fresh, mpc.UseCaseSupportUpdate)
	assert.Same(t, fresh, c.mpcEntity)
}

// ohpcfRemoved is an OHPCF use case that lists no remote entity
type ohpcfRemoved struct{ ucapi.CemOHPCFInterface }

func (ohpcfRemoved) RemoteEntitiesScenarios() []eebusapi.RemoteEntityScenarios { return nil }

// a removed compressor must not be mistaken for a connected one
func TestOHPCFCompressorRemoved(t *testing.T) {
	entity := spinemocks.NewEntityRemoteInterface(t)
	c := &EEBusOHPCF{
		cem:        &eebus.CustomerEnergyManagement{OHPCF: ohpcfRemoved{}},
		connector:  eebus.NewConnector(),
		compressor: entity,
	}

	c.UseCaseEvent(nil, entity, ohpcf.UseCaseSupportUpdate)

	_, ok := c.connectedCompressor()
	assert.False(t, ok)
}
