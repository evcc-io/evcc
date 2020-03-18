package wrapper

import (
	"testing"

	"github.com/golang/mock/gomock"
)

func TestProxyMeterSinglePhase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tc := []struct {
		phases  int64
		voltage float64
		current int64
		power   float64
	}{
		{1, 100, 10, 1000},
		{3, 200, 1, 600},
	}

	for _, tc := range tc {
		m := ChargeMeter{
			Phases:  tc.phases,
			Voltage: tc.voltage,
		}

		m.SetChargeCurrent(tc.current)

		if p, err := m.CurrentPower(); p != tc.power || err != nil {
			t.Errorf("power: %.1f %v", p, err)
		}
	}

}
