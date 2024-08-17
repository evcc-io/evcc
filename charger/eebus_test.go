package charger

import (
	"errors"
	"testing"

	"github.com/enbility/eebus-go/usecases/mocks"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestEEBusIsCharging(t *testing.T) {
	type limitStruct struct {
		phase           uint
		min, max, pause float64
	}

	type measurementStruct struct {
		phase   uint
		current float64
	}

	type testMeasurementStruct struct {
		expected bool
		data     []measurementStruct
	}

	tests := []struct {
		name         string
		limits       []limitStruct
		measurements []testMeasurementStruct
	}{
		{
			"3 phase IEC",
			[]limitStruct{
				{1, 6, 16, 0},
				{2, 6, 16, 0},
				{3, 6, 16, 0},
			},
			[]testMeasurementStruct{
				{
					false,
					[]measurementStruct{
						{1, 0},
						{2, 3},
						{3, 0},
					},
				},
				{
					true,
					[]measurementStruct{
						{1, 6},
						{2, 0},
						{3, 1},
					},
				},
			},
		},
		{
			"1 phase IEC",
			[]limitStruct{
				{1, 6, 16, 0},
			},
			[]testMeasurementStruct{
				{
					false,
					[]measurementStruct{
						{1, 2},
					},
				},
				{
					true,
					[]measurementStruct{
						{1, 6},
					},
				},
			},
		},
		{
			"3 phase ISO",
			[]limitStruct{
				{1, 2.2, 16, 0.1},
				{2, 2.2, 16, 0.1},
				{3, 2.2, 16, 0.1},
			},
			[]testMeasurementStruct{
				{
					false,
					[]measurementStruct{
						{1, 1},
						{2, 0},
						{3, 0},
					},
				},
				{
					true,
					[]measurementStruct{
						{1, 1.8},
						{2, 1},
						{3, 3},
					},
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
					uc: &eebus.UseCasesEVSE{
						EvCC:  evcc,
						EvCem: evcem,
						OpEV:  opev,
					},
					ev: evEntity,
				}

				var currents []float64
				for _, d := range m.data {
					currents = append(currents, d.current)
				}

				evcem.EXPECT().CurrentPerPhase(evEntity).Return(currents, nil)
				opev.EXPECT().CurrentLimits(evEntity).Return(limitsMin, limitsMax, limitsDefault, nil)

				require.Equal(t, m.expected, eebus.isCharging(evEntity))

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
		uc: &eebus.UseCasesEVSE{
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
		uc: &eebus.UseCasesEVSE{
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
