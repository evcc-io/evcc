package eebus

import (
	"testing"
	"time"

	"github.com/enbility/eebus-go/usecases/mocks"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type powerMeter struct{}

func (powerMeter) CurrentPower() (float64, error) { return 0, nil }

type energyMeter struct{ powerMeter }

func (energyMeter) TotalEnergy() (float64, error) { return 0, nil }

type bidirectionalMeter struct{ energyMeter }

func (bidirectionalMeter) ReturnEnergy() (float64, error) { return 0, nil }

type phaseMeter struct{ bidirectionalMeter }

func (phaseMeter) Currents() (float64, float64, float64, error) { return 0, 0, 0, nil }
func (phaseMeter) Voltages() (float64, float64, float64, error) { return 0, 0, 0, nil }

func TestGcpScenarios(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		meter                  api.Meter
		currents, voltages, ok bool
	}{
		{"power only", powerMeter{}, false, false, false},
		{"import only", energyMeter{}, false, false, false},
		{"both directions", bidirectionalMeter{}, false, false, true},
		{"phases", phaseMeter{}, true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			currents, voltages, ok := gcpScenarios(tc.meter)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.currents, currents)
			assert.Equal(t, tc.voltages, voltages)
		})
	}
}

// Phase values without a registered scenario are skipped, their update helpers would dereference a nil config.
func TestGridConnectionPointUpdateWithoutPhases(t *testing.T) {
	power := mocks.NewUpdateData(t)
	power.EXPECT().Supported().Return(true)

	uc := mocks.NewGcpMGCPInterface(t)
	uc.EXPECT().UpdateDataPowerTotal(mock.Anything, mock.Anything, mock.Anything).Return(power)
	uc.EXPECT().Update(mock.Anything).Return(nil)

	g := &GridConnectionPoint{log: util.NewLogger("test"), uc: uc}
	g.update(types.Measurement{Currents: []float64{1, 2, 3}, Voltages: []float64{230, 230, 230}})
}

// Without a fresh measurement the last values are restated once with error state.
func TestGridConnectionPointInvalidate(t *testing.T) {
	data := mocks.NewUpdateData(t)
	data.EXPECT().Supported().Return(true)

	uc := mocks.NewGcpMGCPInterface(t)
	uc.EXPECT().Update(mock.Anything).Return(nil)

	g := &GridConnectionPoint{log: util.NewLogger("test"), uc: uc}

	uc.EXPECT().UpdateDataPowerTotal(-1500.0, mock.Anything, (*model.MeasurementValueStateType)(nil)).Return(data).Once()
	g.update(types.Measurement{Power: -1500})

	g.invalidate(time.Minute)

	g.updated = time.Now().Add(-time.Hour)
	uc.EXPECT().UpdateDataPowerTotal(-1500.0, mock.Anything, mock.MatchedBy(func(s *model.MeasurementValueStateType) bool {
		return s != nil && *s == model.MeasurementValueStateTypeError
	})).Return(data).Once()
	g.invalidate(time.Minute)

	// stale state is only written once
	g.invalidate(time.Minute)
}
