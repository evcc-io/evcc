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
    //9 kWh userBatCap => 10 kWh virtualBatCap
	vehicle.EXPECT().Capacity().Return(int64(9))

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
    //9 kWh userBatCap => 10 kWh virtualBatCap
	vehicle.EXPECT().Capacity().Return(int64(9))

	ce := NewSocEstimator(util.NewLogger("foo"), vehicle, true)
	ce.socCharge = 20.0

	tc := []struct {
		chargedEnergy float64
		vehicleSoC    float64
		estimatedSoC  float64
	}{
		{10, 20.0, 20.0},
		{0, 20.0, 20.0},
		{123, 20.0, 21.23},
		{1000, 20.0, 30.0},
		{1100, 31.0, 31.0},
		{1200, 32.0, 32.0},
		{1900, 39.0, 39.0},
		{2000, 40.0, 40.0},
		{4000, 50.0, 50.0},
		{6000, 60.0, 60.0},
		{6500, 65.0, 65.0},
		{7000, 65.0, 70.0},
		{7100, 71.0, 71.0},
		{7300, 72.0, 72.0},
		{7400, 73.0, 73.0},
		{7700, 75.0, 75.0},
		{8200, 80.0, 80.0},
		{0, 25.0, 25.0},
		{2500, 25.0, 50.0},
		{0, 50.0, 50.0}, // -10000
		{4990, 50.0, 99.9},
		{5000, 50.0, 100.0},
		{5001, 50.0, 100.0},
		{0, 0.0, 0.0},
		{1000, 0.0, 10.0},
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
