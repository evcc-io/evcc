package soc

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRemainingChargeDuration(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)
	vehicle := api.NewMockVehicle(ctrl)
	// 8.5 kWh userBatCap => 10 kWh virtualBatCap (at 85% efficiency)
	vehicle.EXPECT().Capacity().Return(float64(8.5))

	ce := NewEstimator(util.NewLogger("foo"), charger, vehicle)
	ce.vehicleSoc = 20.0

	chargePower := 1000.0
	targetSoc := 80.0

	if remaining := ce.RemainingChargeDuration(targetSoc, chargePower); remaining != 6*time.Hour {
		t.Errorf("wrong remaining charge duration: %v", remaining)
	}
}

func TestSocEstimation(t *testing.T) {
	type chargerStruct struct {
		*api.MockCharger
		*api.MockBattery
	}

	ctrl := gomock.NewController(t)
	vehicle := api.NewMockVehicle(ctrl)
	charger := &chargerStruct{api.NewMockCharger(ctrl), api.NewMockBattery(ctrl)}

	// 8.5 kWh user battery capacity is converted to initial value of 10 kWh virtual capacity (at 85% efficiency)
	vehicle.EXPECT().Capacity().Return(8.5).AnyTimes()

	ce := NewEstimator(util.NewLogger("foo"), charger, vehicle)

	tc := []struct {
		chargedEnergy   float64
		vehicleSoc      float64
		estimatedSoc    float64
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
		{1000, 50.0, 50.0, 8500}, // cap virtual capacity minimum to physical capacity
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)

		soc := ce.Soc(&tc.vehicleSoc, tc.chargedEnergy)

		// validate soc/capacity estimate
		assert.Equal(t, tc.estimatedSoc, soc, "estimated soc")
		assert.Equal(t, tc.virtualCapacity, ce.virtualCapacity, "virtual capacity")

		// validate duration estimate
		chargePower := 1e3
		targetSoc := 100.0
		remainingHours := (float64(targetSoc) - soc) / 100 * tc.virtualCapacity / chargePower
		remainingDuration := time.Duration(float64(time.Hour) * remainingHours).Round(time.Second)

		assert.Equal(t, remainingDuration, ce.RemainingChargeDuration(targetSoc, chargePower), "remaining duration")
	}
}

func TestImprovedEstimatorRemainingChargeDuration(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)
	vehicle := api.NewMockVehicle(ctrl)

	// https://github.com/evcc-io/evcc/pull/7510#issuecomment-1512688548
	// Updated for 85% charge efficiency (previously 90%)
	tc := []struct {
		capacity    float64
		soc         float64
		targetsoc   float64
		chargePower float64
		duration    time.Duration
	}{
		{0.75, 10, 60, 300, 1*time.Hour + 28*time.Minute + 14*time.Second},
		{0.75, 50, 100, 300, 1*time.Hour + 28*time.Minute + 14*time.Second},
		{17, 10, 60, 7 * 1e3, 1*time.Hour + 25*time.Minute + 43*time.Second},
		{17, 50, 100, 7 * 1e3, 1*time.Hour + 33*time.Minute + 35*time.Second},
		{50, 10, 60, 11 * 1e3, 2*time.Hour + 40*time.Minute + 26*time.Second},
		{50, 50, 100, 11 * 1e3, 3*time.Hour + 7*time.Minute + 43*time.Second},
		{80, 10, 60, 22 * 1e3, 2*time.Hour + 8*time.Minute + 21*time.Second},
		{80, 50, 100, 22 * 1e3, 2*time.Hour + 58*time.Minute + 34*time.Second},
	}

	for _, tc := range tc {
		t.Log(tc)

		vehicle.EXPECT().Capacity().Return(tc.capacity)

		ce := NewEstimator(util.NewLogger("foo"), charger, vehicle)
		ce.vehicleSoc = tc.soc

		assert.Equal(t, tc.duration, ce.RemainingChargeDuration(tc.targetsoc, tc.chargePower))
	}
}
