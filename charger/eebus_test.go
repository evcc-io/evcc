package charger

import (
	"errors"
	"testing"
	"time"

	evcemuc "github.com/enbility/eebus-go/usecases/cem/evcem"
	"github.com/enbility/eebus-go/usecases/mocks"
	spinemocks "github.com/enbility/spine-go/mocks"
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
