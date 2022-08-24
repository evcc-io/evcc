package charger

import (
	"testing"

	"github.com/evcc-io/eebus/communication"
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

func TestEEBusIsCharging(t *testing.T) {
	tests := []struct {
		name         string
		limits       []limitStruct
		measurements []testMeasurementStruct
	}{
		{
			"3 phase IEC",
			[]limitStruct{
				{1, 6.0, 16.0, 0.0},
				{2, 6.0, 16.0, 0.0},
				{3, 6.0, 16.0, 0.0},
			},
			[]testMeasurementStruct{
				{
					false,
					[]measurementStruct{
						{1, 0.0},
						{2, 3.0},
						{3, 0.0},
					},
				},
				{
					true,
					[]measurementStruct{
						{1, 6.0},
						{2, 0.0},
						{3, 1.0},
					},
				},
			},
		},
		{
			"1 phase IEC",
			[]limitStruct{
				{1, 6.0, 16.0, 0.0},
			},
			[]testMeasurementStruct{
				{
					false,
					[]measurementStruct{
						{1, 2.0},
					},
				},
				{
					true,
					[]measurementStruct{
						{1, 6.0},
					},
				},
			},
		},
		{
			"3 phase ISO",
			[]limitStruct{
				{1, 2.2, 16.0, 0.1},
				{2, 2.2, 16.0, 0.1},
				{3, 2.2, 16.0, 0.1},
			},
			[]testMeasurementStruct{
				{
					false,
					[]measurementStruct{
						{1, 1.0},
						{2, 0.0},
						{3, 0.0},
					},
				},
				{
					true,
					[]measurementStruct{
						{1, 1.8},
						{2, 1.0},
						{3, 3.0},
					},
				},
			},
		},
	}

	eebus := &EEBus{}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := &communication.EVSEClientDataType{
				EVData: communication.EVDataType{
					ConnectedPhases: 3,
					Limits:          make(map[uint]communication.EVCurrentLimitType),
					Measurements:    communication.EVMeasurementsType{},
				},
			}

			for _, limit := range tc.limits {
				data.EVData.Limits[limit.phase] = communication.EVCurrentLimitType{
					Min:     limit.min,
					Max:     limit.max,
					Default: limit.pause,
				}
			}

			for index, m := range tc.measurements {
				for _, d := range m.data {
					data.EVData.Measurements.Current.Store(d.phase, d.current)
				}

				result := eebus.isCharging(data)
				if result != m.expected {
					t.Errorf("Failure: test %s, series %d, expected %v, got %v", tc.name, index, m.expected, result)
				}
			}
		})
	}
}
