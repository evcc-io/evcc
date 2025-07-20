package wrapper

import (
	"testing"

	"go.uber.org/mock/gomock"
)

func TestProxyChargeMeter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tc := []float64{600, 1000, 2000}
	m := ChargeMeter{}

	for _, f := range tc {
		m.SetPower(f)

		if p, err := m.CurrentPower(); p != f || err != nil {
			t.Errorf("power: %.1f %v", p, err)
		}
	}
}
