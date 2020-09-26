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

	// 9 kWh user battery capacity is converted to initial value of 10 kWh virtual capacity
	var capacity int64 = 9
	vehicle.EXPECT().Capacity().Return(capacity)

	ce := NewSocEstimator(util.NewLogger("foo"), vehicle, true)
	ce.socCharge = 20.0

	tc := []struct {
		chargedEnergy   float64
		vehicleSoC      float64
		estimatedSoC    float64
		virtualCapacity float64
	}{
		{10, 20.0, 20.0, 10000},
		{0, 20.0, 20.0, 10000},
		{123, 20.0, 21.23, 10000},
		{1000, 20.0, 30.0, 10000},
		{1100, 31.0, 31.0, 10000},
		{1200, 32.0, 32.0, 10000},
		{1900, 39.0, 39.0, 10000},
		{2000, 40.0, 40.0, 10000},
		{4000, 50.0, 50.0, 20000}, // @Uli?
		{6000, 60.0, 60.0, 20000}, // @Uli?
		{6500, 65.0, 65.0, 10000},
		{7000, 65.0, 70.0, 10000},
		{7100, 71.0, 71.0, 10000},
		{7300, 72.0, 72.0, 10000},
		{7400, 73.0, 73.0, 10000},
		{7700, 75.0, 75.0, 15000}, // @Uli?
		{8200, 80.0, 80.0, 10000},
		{0, 25.0, 25.0, 10000},
		{2500, 25.0, 50.0, 10000},
		{0, 50.0, 50.0, 10000}, // -10000
		{4990, 50.0, 99.9, 10000},
		{5000, 50.0, 100.0, 10000},
		{5001, 50.0, 100.0, 10000},
		{0, 0.0, 0.0, 10000},
		{1000, 0.0, 10.0, 10000},
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)
		vehicle.EXPECT().ChargeState().Return(tc.vehicleSoC, nil)

		soc, err := ce.SoC(tc.chargedEnergy)
		if err != nil {
			t.Error(err)
		}

		// validate soc estimate
		if tc.estimatedSoC != soc {
			t.Errorf("expected estimated soc: %g, got: %g (virtual capacity: %v)", tc.estimatedSoC, soc, ce.virtualCapacity)
		}

		// validate capacity estimate
		if tc.virtualCapacity != ce.virtualCapacity {
			t.Errorf("expected virtual capacity: %v, got: %v", tc.virtualCapacity, ce.virtualCapacity)
		}

		// validate duration estimate
		remainingDuration := time.Hour * time.Duration((100-tc.estimatedSoC)/100*tc.virtualCapacity/1e3)

		if rm := ce.RemainingChargeDuration(1e3, 100); rm != remainingDuration {
			t.Errorf("expected estimated duration: %v, got: %v (virtual capacity: %v)", remainingDuration, rm, ce.virtualCapacity)
		}
	}
}
