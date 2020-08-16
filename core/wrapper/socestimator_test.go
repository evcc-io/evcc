package wrapper

import (
	"testing"
	"time"

	"github.com/andig/evcc/mock"
	"github.com/andig/evcc/util"
	"github.com/golang/mock/gomock"
)

func TestRemainingChargeDuration(t *testing.T) {
	ctrl := gomock.NewController(t)
	vehicle := mock.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(int64(10))

	ce := NewSocEstimator(util.NewLogger("foo"), vehicle, false)
	ce.socCharge = 20.0

	chargePower := 1000.0
	targetSoC := 80

	if remaining := ce.RemainingChargeDuration(chargePower, targetSoC); remaining != 6*time.Hour {
		t.Error("wrong remaining charge duration")
	}
}

func TestSoCEstimation(t *testing.T) {
	ctrl := gomock.NewController(t)
	vehicle := mock.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(int64(10))

	ce := NewSocEstimator(util.NewLogger("foo"), vehicle, true)
	ce.socCharge = 20.0

	tc := []struct {
		chargedEnergy float64
		vehicleSoC    float64
		estimatedSoC  float64
	}{
		{2000, 30.0, 30.0},
		{3000, 30.0, 40.0},
		{3500, 30.0, 45.0},
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)
		vehicle.EXPECT().ChargeState().Return(tc.vehicleSoC, nil)

		soc, err := ce.SoC(tc.chargedEnergy)
		if err != nil {
			t.Error(err)
		}
		t.Logf("%+v", ce)

		if tc.estimatedSoC != soc {
			t.Errorf("expected: %g, got: %g", tc.estimatedSoC, soc)
		}
	}
}
