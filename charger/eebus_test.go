package charger

import (
	"testing"

	"github.com/enbility/cemd/emobility"
)

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

// Emobility mock

type EmobilityMock struct {
	connectedPhases                               uint
	currents, limitsMin, limitsMax, limitsDefault []float64
}

func (e *EmobilityMock) EVCurrentChargeState() (emobility.EVChargeStateType, error) {
	return emobility.EVChargeStateTypeUnknown, nil
}

func (e *EmobilityMock) EVConnectedPhases() (uint, error) {
	return e.connectedPhases, nil
}

func (e *EmobilityMock) EVChargedEnergy() (float64, error) {
	return 0, nil
}

func (e *EmobilityMock) EVPowerPerPhase() ([]float64, error) {
	return []float64{}, nil
}

func (e *EmobilityMock) EVCurrentsPerPhase() ([]float64, error) {
	return e.currents, nil
}

func (e *EmobilityMock) EVCurrentLimits() ([]float64, []float64, []float64, error) {
	return e.limitsMin, e.limitsMax, e.limitsDefault, nil
}

func (e *EmobilityMock) EVWriteLoadControlLimits(obligations, recommendations []float64) error {
	return nil
}

func (e *EmobilityMock) EVCommunicationStandard() (emobility.EVCommunicationStandardType, error) {
	return emobility.EVCommunicationStandardTypeUnknown, nil
}

func (e *EmobilityMock) EVIdentification() (string, error) {
	return "", nil
}

func (e *EmobilityMock) EVOptimizationOfSelfConsumptionSupported() (bool, error) {
	return false, nil
}

func (e *EmobilityMock) EVSoCSupported() (bool, error) {
	return false, nil
}

func (e *EmobilityMock) EVSoC() (float64, error) {
	return 0, nil
}

func (e *EmobilityMock) EVCoordinatedChargingSupported() (bool, error) {
	return false, nil
}

var _ emobility.EmobilityI = (*EmobilityMock)(nil)

func TestEEBusIsCharging(t *testing.T) {
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

	emobilityMock := &EmobilityMock{}
	eebus := &EEBus{
		emobility: emobilityMock,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			emobilityMock.connectedPhases = 3
			emobilityMock.limitsMin = make([]float64, 0)
			emobilityMock.limitsMax = make([]float64, 0)
			emobilityMock.limitsDefault = make([]float64, 0)

			for _, limit := range tc.limits {
				emobilityMock.limitsMin = append(emobilityMock.limitsMin, limit.min)
				emobilityMock.limitsMax = append(emobilityMock.limitsMax, limit.max)
				emobilityMock.limitsDefault = append(emobilityMock.limitsDefault, limit.pause)
			}

			for index, m := range tc.measurements {
				emobilityMock.currents = make([]float64, 0)

				for _, d := range m.data {
					emobilityMock.currents = append(emobilityMock.currents, d.current)
				}

				result := eebus.isCharging()
				if result != m.expected {
					t.Errorf("Failure: test %s, series %d, expected %v, got %v", tc.name, index, m.expected, result)
				}
			}
		})
	}
}
