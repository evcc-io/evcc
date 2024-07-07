package charger

import (
	"testing"

	"github.com/enbility/eebus-go/usecases/mocks"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/evcc-io/evcc/charger/eebus"
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
			limitsMin := make([]float64, 0)
			limitsMax := make([]float64, 0)
			limitsDefault := make([]float64, 0)

			for _, limit := range tc.limits {
				limitsMin = append(limitsMin, limit.min)
				limitsMax = append(limitsMax, limit.max)
				limitsDefault = append(limitsDefault, limit.pause)
			}

			for index, m := range tc.measurements {
				ctrl := gomock.NewController(t)

				evcc := mocks.NewCemEVCCInterface(t)
				evcem := mocks.NewCemEVCEMInterface(t)
				opev := mocks.NewCemOPEVInterface(t)

				uc := &eebus.UseCasesEVSE{
					EvCC:  evcc,
					EvCem: evcem,
					OpEV:  opev,
				}
				evEntity := spinemocks.NewEntityRemoteInterface(t)
				eebus := &EEBus{
					uc: uc,
					ev: evEntity,
				}

				currents := make([]float64, 0)

				for _, d := range m.data {
					currents = append(currents, d.current)
				}

				evcc.EXPECT().EVConnected(evEntity).Return(true)
				evcem.EXPECT().CurrentPerPhase(evEntity).Return(currents, nil)
				opev.EXPECT().CurrentLimits(evEntity).Return(limitsMin, limitsMax, limitsDefault, nil)

				result := eebus.isCharging()
				if result != m.expected {
					t.Errorf("Failure: test %s, series %d, expected %v, got %v", tc.name, index, m.expected, result)
				}
				ctrl.Finish()
			}
		})
	}
}
