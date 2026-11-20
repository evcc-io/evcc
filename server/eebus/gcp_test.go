package eebus

import (
	"testing"

	"github.com/enbility/eebus-go/usecases/mocks"
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
	g.Update(types.Measurement{Currents: []float64{1, 2, 3}, Voltages: []float64{230, 230, 230}})
}
