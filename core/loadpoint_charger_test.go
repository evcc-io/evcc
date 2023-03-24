package core

import (
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/wrapper"
	"github.com/evcc-io/evcc/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestChargerWrapper(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockCharger := mock.NewMockCharger(ctrl)
	mockMeter := mock.NewMockMeter(ctrl)
	mockMeterEnergy := mock.NewMockMeterEnergy(ctrl)
	mockPhaseCurrents := mock.NewMockPhaseCurrents(ctrl)

	{
		lp := &Loadpoint{
			bus: evbus.New(),
		}

		lp.configureChargerType(mockCharger)
		_, ok := lp.chargeMeter.(*wrapper.ChargeMeter)
		assert.True(t, ok)

		_, ok = lp.chargeMeter.(api.Meter)
		assert.True(t, ok)
		assert.Implements(t, (*api.Meter)(nil), lp.chargeMeter)
	}

	{
		lp := &Loadpoint{
			bus: evbus.New(),
		}

		charger := struct {
			api.Charger
			api.Meter
		}{
			mockCharger,
			mockMeter,
		}

		lp.configureChargerType(charger)
		_, ok := lp.chargeMeter.(*wrapper.ChargeMeter)
		assert.False(t, ok)

		_, ok = lp.chargeMeter.(api.Meter)
		assert.True(t, ok)
		assert.Implements(t, (*api.Meter)(nil), lp.chargeMeter)
	}

	{
		lp := &Loadpoint{
			bus: evbus.New(),
		}

		charger := struct {
			api.Charger
			api.Meter
			api.MeterEnergy
		}{
			mockCharger,
			mockMeter,
			mockMeterEnergy,
		}

		lp.configureChargerType(charger)
		_, ok := lp.chargeMeter.(*wrapper.ChargeMeter)
		assert.False(t, ok)

		_, ok = lp.chargeMeter.(api.Meter)
		assert.True(t, ok)
		assert.Implements(t, (*api.Meter)(nil), lp.chargeMeter)
		assert.Implements(t, (*api.MeterEnergy)(nil), lp.chargeMeter)
	}

	{
		lp := &Loadpoint{
			bus: evbus.New(),
		}

		charger := struct {
			api.Charger
			api.PhaseCurrents
			api.MeterEnergy
		}{
			mockCharger,
			mockPhaseCurrents,
			mockMeterEnergy,
		}

		lp.configureChargerType(charger)
		_, ok := lp.chargeMeter.(*wrapper.ChargePhaseMeter)
		assert.True(t, ok)

		_, ok = lp.chargeMeter.(api.Meter)
		assert.True(t, ok)
		assert.Implements(t, (*api.Meter)(nil), lp.chargeMeter)
		assert.Implements(t, (*api.MeterEnergy)(nil), lp.chargeMeter)
	}
}
