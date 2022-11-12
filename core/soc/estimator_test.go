package soc

import (
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/mock"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
)

func TestRemainingChargeDuration(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := mock.NewMockCharger(ctrl)
	vehicle := mock.NewMockVehicle(ctrl)
	// 9 kWh userBatCap => 10 kWh virtualBatCap
	vehicle.EXPECT().Capacity().Return(float64(9))

	ce := NewEstimator(util.NewLogger("foo"), charger, vehicle, false)
	ce.vehicleSoc = 20.0

	chargePower := 1000.0
	targetSoC := 80

	if remaining := ce.RemainingChargeDuration(chargePower, targetSoC); remaining != 6*time.Hour {
		t.Error("wrong remaining charge duration")
	}
}

func TestSoCEstimation(t *testing.T) {
	type chargerStruct struct {
		*mock.MockCharger
		*mock.MockBattery
	}

	ctrl := gomock.NewController(t)
	vehicle := mock.NewMockVehicle(ctrl)
	charger := &chargerStruct{mock.NewMockCharger(ctrl), mock.NewMockBattery(ctrl)}

	// 9 kWh user battery capacity is converted to initial value of 10 kWh virtual capacity
	var capacity float64 = 9
	vehicle.EXPECT().Capacity().Return(capacity)

	ce := NewEstimator(util.NewLogger("foo"), charger, vehicle, true)
	ce.vehicleSoc = 0.0

	tc := []struct {
		chargedEnergy   float64
		vehicleSoC      float64
		estimatedSoC    float64
		virtualCapacity float64
	}{
		{0, 0.0, 0.0, 10000},
		{0, 20.0, 20.0, 10000},
		{123, 20.0, 21.23, 10000},
		{1000, 20.0, 30.0, 10000},
		{1100, 31.0, 31.0, 10000},
		{1200, 32.0, 32.0, 10000},
		{1900, 39.0, 39.0, 10000},
		{2000, 40.0, 40.0, 10000},
		{6000, 80.0, 80.0, 10000},
		{0, 25.0, 25.0, 10000},
		{2500, 25.0, 50.0, 10000},
		{0, 50.0, 50.0, 10000}, // -10000
		{4990, 50.0, 99.9, 10000},
		{5000, 50.0, 100.0, 10000},
		{5001, 50.0, 100.0, 10000},
		{0, 20.0, 20.0, 10000},
		{1000, 30.0, 30.0, 10000},
	}

	for i := 1; i < 3; i++ {
		useVehicleSoC := true
		if i == 2 {
			useVehicleSoC = false
		}
		for _, tc := range tc {
			t.Logf("%+v", tc)
			if useVehicleSoC {
				charger.MockBattery.EXPECT().SoC().Return(tc.vehicleSoC, nil)
			} else {
				charger.MockBattery.EXPECT().SoC().Return(0.0, api.ErrNotAvailable)
				vehicle.EXPECT().SoC().Return(tc.vehicleSoC, nil)
			}

			soc, err := ce.SoC(tc.chargedEnergy)
			if err != nil {
				t.Error(err)
			}

			// validate soc estimate
			if tc.estimatedSoC != soc {
				t.Errorf("expected estimated soc: %g, got: %g", tc.estimatedSoC, soc)
			}

			// validate capacity estimate
			if tc.virtualCapacity != ce.virtualCapacity {
				t.Errorf("expected virtual capacity: %v, got: %v", tc.virtualCapacity, ce.virtualCapacity)
			}

			// validate duration estimate
			chargePower := 1e3
			targetSoC := 100
			remainingHours := (float64(targetSoC) - soc) / 100 * tc.virtualCapacity / chargePower
			remainingDuration := time.Duration(float64(time.Hour) * remainingHours).Round(time.Second)

			if rm := ce.RemainingChargeDuration(chargePower, targetSoC); rm != remainingDuration {
				t.Errorf("expected estimated duration: %v, got: %v", remainingDuration, rm)
			}
		}
	}
}

func TestSoCFromChargerAndVehicleWithErrors(t *testing.T) {
	type chargerStruct struct {
		*mock.MockCharger
		*mock.MockBattery
	}

	ctrl := gomock.NewController(t)
	vehicle := mock.NewMockVehicle(ctrl)
	charger := &chargerStruct{mock.NewMockCharger(ctrl), mock.NewMockBattery(ctrl)}

	// 9 kWh user battery capacity is converted to initial value of 10 kWh virtual capacity
	var capacity float64 = 9
	vehicle.EXPECT().Capacity().Return(capacity)

	ce := NewEstimator(util.NewLogger("foo"), charger, vehicle, true)
	ce.vehicleSoc = 20.0

	tc := []struct {
		chargedEnergy   float64
		vehicleSoC      float64
		estimatedSoC    float64
		virtualCapacity float64
		expectVehicle   bool
		chargerError    error
		vehicleError    error
	}{
		// start with SoC from charger and errors
		{0, 0.0, 0.0, 10000, false, errors.New("some error"), nil},
		{0, 0.0, 20.0, 10000, false, api.ErrMustRetry, nil},
		{0, 20.0, 20.0, 10000, false, nil, nil},
		{123, 20.0, 21.23, 10000, false, nil, nil},
		{123, 0.0, 21.23, 10000, false, errors.New("another error"), nil},
		{1000, 20.0, 30.0, 10000, false, nil, nil},
		{1100, 31.0, 31.0, 10000, false, nil, nil},
		{1200, 32.0, 32.0, 10000, false, nil, nil},
		{1900, 39.0, 39.0, 10000, false, nil, nil},
		{2000, 40.0, 40.0, 10000, false, nil, nil},
		// move to SoC from vehicle
		{3000, 0.0, 50.0, 10000, true, api.ErrNotAvailable, errors.New("some error")},
		{3100, 0.0, 51.0, 10000, true, api.ErrNotAvailable, api.ErrMustRetry},
		{5100, 71.0, 71.0, 10000, true, api.ErrNotAvailable, nil},
		{5200, 72.0, 72.0, 10000, true, api.ErrNotAvailable, nil},
		{5300, 0.0, 73.0, 10000, true, api.ErrNotAvailable, errors.New("another error")},
		{5300, 73.0, 73.0, 10000, true, api.ErrNotAvailable, nil},
		{5500, 75.0, 75.0, 10000, true, api.ErrNotAvailable, nil},
		{6000, 80.0, 80.0, 10000, true, api.ErrNotAvailable, nil},
		{0, 25.0, 25.0, 10000, true, api.ErrNotAvailable, nil},
		{2500, 25.0, 50.0, 10000, true, api.ErrNotAvailable, nil},
		{0, 50.0, 50.0, 10000, true, api.ErrNotAvailable, nil}, // -10000
		{4990, 50.0, 99.9, 10000, true, api.ErrNotAvailable, nil},
		{5000, 50.0, 100.0, 10000, true, api.ErrNotAvailable, nil},
		// back to SoC from charger
		{5001, 50.0, 100.0, 10000, false, nil, nil},
		{0, 20.0, 20.0, 10000, false, nil, nil},
		{1000, 30.0, 30.0, 10000, false, nil, nil},
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)
		charger.MockBattery.EXPECT().SoC().Return(tc.vehicleSoC, tc.chargerError)
		if tc.expectVehicle {
			vehicle.EXPECT().SoC().Return(tc.vehicleSoC, tc.vehicleError)
		}

		soc, err := ce.SoC(tc.chargedEnergy)
		if err != nil {
			if (!tc.expectVehicle && err != tc.chargerError) || (tc.expectVehicle && err != tc.vehicleError) {
				t.Error(err)
			} else {
				continue
			}
		}

		// validate soc estimate
		if tc.estimatedSoC != soc {
			t.Errorf("expected estimated soc: %g, got: %g", tc.estimatedSoC, soc)
		}

		// validate capacity estimate
		if tc.virtualCapacity != ce.virtualCapacity {
			t.Errorf("expected virtual capacity: %v, got: %v", tc.virtualCapacity, ce.virtualCapacity)
		}

		// validate duration estimate
		chargePower := 1e3
		targetSoC := 100
		remainingHours := (float64(targetSoC) - soc) / 100 * tc.virtualCapacity / chargePower
		remainingDuration := time.Duration(float64(time.Hour) * remainingHours).Round(time.Second)

		if rm := ce.RemainingChargeDuration(chargePower, targetSoC); rm != remainingDuration {
			t.Errorf("expected estimated duration: %v, got: %v", remainingDuration, rm)
		}
	}
}
