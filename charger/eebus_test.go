package charger

import (
	"errors"
	"testing"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	evcemuc "github.com/enbility/eebus-go/usecases/cem/evcem"
	"github.com/enbility/eebus-go/usecases/mocks"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Test measurements updated after writing limits detction works
func TestEEBusNoCurrents(t *testing.T) {
	evcc := mocks.NewCemEVCCInterface(t)
	evcem := mocks.NewCemEVCEMInterface(t)

	evEntity := spinemocks.NewEntityRemoteInterface(t)
	eebus := &EEBus{
		cem: &eebus.CustomerEnergyManagement{
			EvCC:  evcc,
			EvCem: evcem,
		},
		ev:  evEntity,
		log: util.NewLogger("test"),
	}

	evcc.EXPECT().EVConnected(evEntity).Return(true)
	evcem.EXPECT().IsScenarioAvailableAtEntity(evEntity, mock.Anything).Return(true)

	// limit set 15:04:45, measurement receviced afterwards before calling currents
	eebus.limitUpdated = time.Date(2024, 9, 16, 15, 4, 45, 0, time.UTC)
	eebus.UseCaseEvent(nil, evEntity, evcemuc.DataUpdateCurrentPerPhase)

	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{10.5, 10.5, 10.5}, nil).Once()

	l1, l2, l3, err := eebus.currents()
	require.NoError(t, err)
	assert.Equal(t, 10.5, l1)
	assert.Equal(t, 10.5, l2)
	assert.Equal(t, 10.5, l3)

	// limit set 15:05:09, measurement receviced afterwards before calling currents
	eebus.limitUpdated = time.Date(2024, 9, 16, 15, 5, 9, 0, time.UTC)
	eebus.UseCaseEvent(nil, evEntity, evcemuc.DataUpdateCurrentPerPhase)

	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{6.6, 6.6, 6.6}, nil).Once()

	l1, l2, l3, err = eebus.currents()
	require.NoError(t, err)
	assert.Equal(t, 6.6, l1)
	assert.Equal(t, 6.6, l2)
	assert.Equal(t, 6.6, l3)

	// limit set 15:05:39, measurement received afterwards before calling currents
	eebus.limitUpdated = time.Date(2024, 9, 16, 15, 5, 39, 0, time.UTC)
	eebus.UseCaseEvent(nil, evEntity, evcemuc.DataUpdateCurrentPerPhase)

	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{10.4, 10.5, 10.4}, nil).Once()

	l1, l2, l3, err = eebus.currents()
	require.NoError(t, err)
	assert.Equal(t, 10.4, l1)
	assert.Equal(t, 10.5, l2)
	assert.Equal(t, 10.4, l3)

	// limit set 15:06:09, measurement received afterwards before calling currents
	eebus.limitUpdated = time.Date(2024, 9, 16, 15, 6, 9, 0, time.UTC)
	eebus.UseCaseEvent(nil, evEntity, evcemuc.DataUpdateCurrentPerPhase)

	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{10.4, 10.4, 10.4}, nil).Once()

	l1, l2, l3, err = eebus.currents()
	require.NoError(t, err)
	assert.Equal(t, 10.4, l1)
	assert.Equal(t, 10.4, l2)
	assert.Equal(t, 10.4, l3)

	// limit set 20 seconds ago, no measurement received yet
	eebus.limitUpdated = time.Now().Add(-20 * time.Second)

	l1, l2, l3, err = eebus.currents()
	require.Error(t, err)
	assert.Equal(t, 0.0, l1)
	assert.Equal(t, 0.0, l2)
	assert.Equal(t, 0.0, l3)

	// now we got a measurement again
	eebus.UseCaseEvent(nil, evEntity, evcemuc.DataUpdateCurrentPerPhase)

	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{10.4, 10.4, 10.4}, nil).Once()

	l1, l2, l3, err = eebus.currents()
	require.NoError(t, err)
	assert.Equal(t, 10.4, l1)
	assert.Equal(t, 10.4, l2)
	assert.Equal(t, 10.4, l3)
}

// newTestEEBus creates an EEBus instance with all mocks wired up for limit writing tests.
func newTestEEBus(t *testing.T) (*EEBus, *mocks.CemOPEVInterface, *mocks.CemOSCEVInterface, *spinemocks.EntityRemoteInterface) {
	t.Helper()

	opev := mocks.NewCemOPEVInterface(t)
	oscev := mocks.NewCemOSCEVInterface(t)
	evEntity := spinemocks.NewEntityRemoteInterface(t)

	eebus := &EEBus{
		cem: &eebus.CustomerEnergyManagement{
			OpEV:  opev,
			OscEV: oscev,
		},
		ev:  evEntity,
		log: util.NewLogger("test"),
	}

	return eebus, opev, oscev, evEntity
}

// 3-phase limits helper
func opevLimits3p(min, max, def float64) ([]float64, []float64, []float64, error) {
	return []float64{min, min, min}, []float64{max, max, max}, []float64{def, def, def}, nil
}

func TestWriteCurrentLimitData_OpevOnly(t *testing.T) {
	eebus, opev, oscev, evEntity := newTestEEBus(t)
	_ = eebus

	// OPEV available, OSCEV not available
	opev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
	opev.EXPECT().CurrentLimits(evEntity).Return(opevLimits3p(6, 16, 0))
	opev.EXPECT().WriteLoadControlLimits(evEntity, mock.MatchedBy(func(limits []ucapi.LoadLimitsPhase) bool {
		return len(limits) == 3 && limits[0].IsActive && limits[0].Value == 10
	}), mock.Anything).Return(nil, nil)

	oscev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(false)

	err := eebus.writeCurrentLimitData(evEntity, 10)
	require.NoError(t, err)
}

func TestWriteCurrentLimitData_OpevAndOscev(t *testing.T) {
	eebus, opev, oscev, evEntity := newTestEEBus(t)
	_ = eebus

	// Both available, current = 10A (between min and max)
	opev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
	opev.EXPECT().CurrentLimits(evEntity).Return(opevLimits3p(6, 16, 0))
	opev.EXPECT().WriteLoadControlLimits(evEntity, mock.MatchedBy(func(limits []ucapi.LoadLimitsPhase) bool {
		// OPEV: active at 10A (below max of 16)
		return len(limits) == 3 && limits[0].IsActive && limits[0].Value == 10
	}), mock.Anything).Return(nil, nil)

	oscev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
	oscev.EXPECT().LoadControlLimits(evEntity).Return([]ucapi.LoadLimitsPhase{}, nil)
	oscev.EXPECT().CurrentLimits(evEntity).Return(opevLimits3p(2, 16, 0))
	oscev.EXPECT().WriteLoadControlLimits(evEntity, mock.MatchedBy(func(limits []ucapi.LoadLimitsPhase) bool {
		// OSCEV: active at 10A (>= min of 2, recommendation to charge)
		return len(limits) == 3 && limits[0].IsActive && limits[0].Value == 10
	}), mock.Anything).Return(nil, nil)

	err := eebus.writeCurrentLimitData(evEntity, 10)
	require.NoError(t, err)
}

func TestWriteCurrentLimitData_AtMax(t *testing.T) {
	eebus, opev, oscev, evEntity := newTestEEBus(t)
	_ = eebus

	// Current equals max limit
	opev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
	opev.EXPECT().CurrentLimits(evEntity).Return(opevLimits3p(6, 16, 0))
	opev.EXPECT().WriteLoadControlLimits(evEntity, mock.MatchedBy(func(limits []ucapi.LoadLimitsPhase) bool {
		// OPEV: inactive at max (no restriction needed)
		return len(limits) == 3 && !limits[0].IsActive && limits[0].Value == 16
	}), mock.Anything).Return(nil, nil)

	oscev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
	oscev.EXPECT().LoadControlLimits(evEntity).Return([]ucapi.LoadLimitsPhase{}, nil)
	oscev.EXPECT().CurrentLimits(evEntity).Return(opevLimits3p(2, 16, 0))
	oscev.EXPECT().WriteLoadControlLimits(evEntity, mock.MatchedBy(func(limits []ucapi.LoadLimitsPhase) bool {
		// OSCEV: active at 16A (>= min, recommend charging)
		return len(limits) == 3 && limits[0].IsActive && limits[0].Value == 16
	}), mock.Anything).Return(nil, nil)

	err := eebus.writeCurrentLimitData(evEntity, 16)
	require.NoError(t, err)
}

func TestWriteCurrentLimitData_Disable(t *testing.T) {
	eebus, opev, oscev, evEntity := newTestEEBus(t)
	_ = eebus

	// Current = 0 (disable charging)
	opev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
	opev.EXPECT().CurrentLimits(evEntity).Return(opevLimits3p(6, 16, 0))
	opev.EXPECT().WriteLoadControlLimits(evEntity, mock.MatchedBy(func(limits []ucapi.LoadLimitsPhase) bool {
		// OPEV: active at 0A (hard stop)
		return len(limits) == 3 && limits[0].IsActive && limits[0].Value == 0
	}), mock.Anything).Return(nil, nil)

	oscev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
	oscev.EXPECT().LoadControlLimits(evEntity).Return([]ucapi.LoadLimitsPhase{}, nil)
	oscev.EXPECT().CurrentLimits(evEntity).Return(opevLimits3p(2, 16, 0))
	oscev.EXPECT().WriteLoadControlLimits(evEntity, mock.MatchedBy(func(limits []ucapi.LoadLimitsPhase) bool {
		// OSCEV: inactive at 0A (no recommendation, < min)
		return len(limits) == 3 && !limits[0].IsActive && limits[0].Value == 0
	}), mock.Anything).Return(nil, nil)

	err := eebus.writeCurrentLimitData(evEntity, 0)
	require.NoError(t, err)
}

func TestWriteCurrentLimitData_OscevNoLimitData(t *testing.T) {
	eebus, opev, oscev, evEntity := newTestEEBus(t)
	_ = eebus

	// OSCEV scenario available but no limit data (e.g. PCMP wallbox)
	opev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
	opev.EXPECT().CurrentLimits(evEntity).Return(opevLimits3p(6, 16, 0))
	opev.EXPECT().WriteLoadControlLimits(evEntity, mock.Anything, mock.Anything).Return(nil, nil)

	oscev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
	oscev.EXPECT().LoadControlLimits(evEntity).Return(nil, errors.New("data not available"))
	// no WriteLoadControlLimits call expected for OSCEV

	err := eebus.writeCurrentLimitData(evEntity, 10)
	require.NoError(t, err)
}

func TestWriteCurrentLimitData_OpevNotAvailable(t *testing.T) {
	eebus, opev, _, evEntity := newTestEEBus(t)
	_ = eebus

	opev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(false)

	err := eebus.writeCurrentLimitData(evEntity, 10)
	require.ErrorIs(t, err, api.ErrNotAvailable)
}

func TestEnabledAlwaysReadsOpev(t *testing.T) {
	evcc := mocks.NewCemEVCCInterface(t)
	opev := mocks.NewCemOPEVInterface(t)

	evEntity := spinemocks.NewEntityRemoteInterface(t)
	eebus := &EEBus{
		cem: &eebus.CustomerEnergyManagement{
			EvCC: evcc,
			OpEV: opev,
		},
		ev:  evEntity,
		log: util.NewLogger("test"),
	}

	evcc.EXPECT().EVConnected(evEntity).Return(true)
	opev.EXPECT().LoadControlLimits(evEntity).Return([]ucapi.LoadLimitsPhase{
		{Phase: model.ElectricalConnectionPhaseNameTypeA, IsActive: true, Value: 10},
		{Phase: model.ElectricalConnectionPhaseNameTypeB, IsActive: true, Value: 10},
		{Phase: model.ElectricalConnectionPhaseNameTypeC, IsActive: true, Value: 10},
	}, nil)

	enabled, err := eebus.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)
}

func TestEEBusIsCharging(t *testing.T) {
	type limitStruct struct {
		min, max, pause float64
	}

	type testMeasurementStruct struct {
		charging bool
		currents []float64
		powers   []float64
	}

	tests := []struct {
		name         string
		limits       []limitStruct
		measurements []testMeasurementStruct
	}{
		{
			"3 phase IEC",
			[]limitStruct{
				{6, 16, 0},
				{6, 16, 0},
				{6, 16, 0},
			},
			[]testMeasurementStruct{
				{
					false,
					[]float64{0, 3, 0},
					[]float64{0, 690, 0},
				},
				{
					true,
					[]float64{6, 0, 1},
					[]float64{1380, 0, 230},
				},
			},
		},
		{
			"1 phase IEC",
			[]limitStruct{
				{6, 16, 0},
			},
			[]testMeasurementStruct{
				{
					false,
					[]float64{2},
					[]float64{460},
				},
				{
					true,
					[]float64{6},
					[]float64{1380},
				},
			},
		},
		{
			"3 phase ISO",
			[]limitStruct{
				{2.2, 16, 0.1},
				{2.2, 16, 0.1},
				{2.2, 16, 0.1},
			},
			[]testMeasurementStruct{
				{
					false,
					[]float64{1, 0, 0},
					[]float64{230, 0, 0},
				},
				{
					true,
					[]float64{1.8, 1, 3},
					[]float64{414, 230, 690},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var limitsMin, limitsMax, limitsDefault []float64

			for _, limit := range tc.limits {
				limitsMin = append(limitsMin, limit.min)
				limitsMax = append(limitsMax, limit.max)
				limitsDefault = append(limitsDefault, limit.pause)
			}

			for _, m := range tc.measurements {
				ctrl := gomock.NewController(t)

				evcc := mocks.NewCemEVCCInterface(t)
				evcem := mocks.NewCemEVCEMInterface(t)
				opev := mocks.NewCemOPEVInterface(t)

				evEntity := spinemocks.NewEntityRemoteInterface(t)
				eebus := &EEBus{
					cem: &eebus.CustomerEnergyManagement{
						EvCC:  evcc,
						EvCem: evcem,
						OpEV:  opev,
					},
					ev: evEntity,
				}

				evcc.EXPECT().EVConnected(evEntity).Return(true)
				evcem.EXPECT().IsScenarioAvailableAtEntity(evEntity, mock.Anything).Return(true)
				evcem.EXPECT().PowerPerPhase(evEntity).Return(m.powers, nil)
				opev.EXPECT().CurrentLimits(evEntity).Return(limitsMin, limitsMax, limitsDefault, nil)

				require.Equal(t, m.charging, eebus.isCharging(evEntity))

				ctrl.Finish()
			}
		})
	}
}

func TestEEBusCurrentPower(t *testing.T) {
	evcc := mocks.NewCemEVCCInterface(t)
	evcem := mocks.NewCemEVCEMInterface(t)

	evEntity := spinemocks.NewEntityRemoteInterface(t)
	eebus := &EEBus{
		cem: &eebus.CustomerEnergyManagement{
			EvCC:  evcc,
			EvCem: evcem,
		},
		ev:  evEntity,
		log: util.NewLogger("test"),
	}

	evcc.EXPECT().EVConnected(evEntity).Return(true)
	evcem.EXPECT().IsScenarioAvailableAtEntity(evEntity, mock.Anything).Return(true)
	evcem.EXPECT().PowerPerPhase(evEntity).Return([]float64{600, 600, 600}, nil)

	power, err := eebus.currentPower()
	require.NoError(t, err)
	assert.Equal(t, 1800.0, power)
}

func TestEEBusCurrentPower_Elli(t *testing.T) {
	evcc := mocks.NewCemEVCCInterface(t)
	evcem := mocks.NewCemEVCEMInterface(t)

	evEntity := spinemocks.NewEntityRemoteInterface(t)
	eebus := &EEBus{
		cem: &eebus.CustomerEnergyManagement{
			EvCC:  evcc,
			EvCem: evcem,
		},
		ev:  evEntity,
		log: util.NewLogger("test"),
	}

	evcc.EXPECT().EVConnected(evEntity).Return(true)
	evcem.EXPECT().IsScenarioAvailableAtEntity(evEntity, mock.Anything).Return(true)
	evcem.EXPECT().PowerPerPhase(evEntity).Return(nil, errors.New("error"))
	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{5.8, 5.8, 5.8}, nil)

	power, err := eebus.currentPower()
	require.NoError(t, err)
	assert.Equal(t, 4002.0, power)
}
