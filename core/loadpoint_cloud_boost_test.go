package core

import (
	"math"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/grid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type cloudSite struct {
	mockSite
	soc         float64
	coordinated bool
	buffer      float64
}

func (s *cloudSite) CoordinatedPVBattery() bool { return s.coordinated }
func (s *cloudSite) GetBufferSoc() float64      { return s.buffer }

func (s *cloudSite) GetBatterySoc() float64 { return s.soc }

func cloudLoadpoint() (*Loadpoint, *cloudSite) {
	Voltage = 230
	limit := 3300.0
	s := &cloudSite{mockSite: mockSite{maxDischargePower: &limit}, soc: 70}
	lp := &Loadpoint{
		log: util.NewLogger("cloud-test"), clock: clock.NewMock(),
		site: s, charger: plainCharger{}, status: api.StatusC, enabled: true,
		batteryBoost: boostContinue, batteryBoostLimit: 50,
		minCurrent: 6, maxCurrent: 16, phases: 3, phasesConfigured: 3,
		chargePower: 7800,
	}
	return lp, s
}

func TestCloudBoostDefersToOptimizer(t *testing.T) {
	lp, s := cloudLoadpoint()
	s.automatic, s.coordinated = true, true
	lp.vehicle = modelledVehicle(gomock.NewController(t))
	lp.setSuggestion(&types.Suggestion{Action: actionStop})
	require.False(t, lp.usesCloudBoost())
}

func TestCloudBoostReservesAuxiliaryAndFlexiblePower(t *testing.T) {
	lp, _ := cloudLoadpoint()
	s := NewSite()
	res := sitePowerResult{
		power: -4500, flexiblePower: 1000,
		state: siteState{auxPower: 500},
	}
	require.Equal(t, -3000.0, s.loadpointSitePower(lp, res))
}

func TestCloudBoostWeatherSequence(t *testing.T) {
	lp, _ := cloudLoadpoint()
	for _, tc := range []struct{ pv, want float64 }{
		{4500, 7800}, {3000, 6300}, {1500, 4800}, {500, 0},
	} {
		current := lp.pvMaxCurrent(lp.chargePower-tc.pv, 3300, false, false)
		require.InDelta(t, tc.want, grid.CurrentToPower(current, 3), 0.0001)
		if tc.want > 0 {
			lp.chargePower = tc.want
		}
	}
}

func TestCoordinatedPVUsesBufferWithoutBoost(t *testing.T) {
	for _, tc := range []struct{ soc, pv, want float64 }{
		{55, 1500, 4800}, {50, 1500, 0}, {49, 1500, 0}, {50, 4500, 4500},
	} {
		lp, s := cloudLoadpoint()
		s.coordinated, s.buffer, s.soc = true, 50, tc.soc
		lp.batteryBoost, lp.batteryBoostLimit = boostDisabled, 5
		lp.Disable.Delay = 3 * time.Minute
		got := lp.pvMaxCurrent(lp.chargePower-tc.pv, 2700, true, true)
		require.InDelta(t, tc.want, grid.CurrentToPower(got, 3), 0.001)
		require.Equal(t, boostDisabled, lp.GetBatteryBoost())
	}
}

func TestCoordinatedPVNeverStartsFromBufferSoc(t *testing.T) {
	lp, s := cloudLoadpoint()
	s.coordinated, s.buffer, s.soc = true, 50, 90
	lp.batteryBoost = boostDisabled
	lp.enabled, lp.status, lp.chargePower = false, api.StatusB, 0
	require.Zero(t, lp.pvMaxCurrent(-1500, 0, true, true))
	require.Zero(t, lp.pvMaxCurrent(-4139, 0, true, true))
	require.Equal(t, 6.0, lp.pvMaxCurrent(-4140, 0, true, true))
	// An enabled charger waiting for the car is still a start, not a running session.
	lp.enabled = true
	require.Zero(t, lp.pvMaxCurrent(-1500, 0, true, true))
}

func TestCloudBoostZeroLimitPreservesExplicitBoost(t *testing.T) {
	lp, s := cloudLoadpoint()
	lp.batteryBoostLimit = 0
	require.InDelta(t, 4800.0, grid.CurrentToPower(lp.pvMaxCurrent(lp.chargePower-1500, 0, false, false), 3), 0.001)
	s.coordinated, s.buffer = true, 0
	require.Zero(t, lp.pvMaxCurrent(lp.chargePower-1500, 0, false, false))
}

func TestCloudBoostStartAndRestartRequirePV(t *testing.T) {
	lp, _ := cloudLoadpoint()
	lp.enabled, lp.status, lp.chargePower = false, api.StatusB, 0
	lp.batteryBoost = boostStart
	minimum := grid.CurrentToPower(6, 3)
	require.Zero(t, lp.pvMaxCurrent(-(minimum-1), 0, true, true))
	require.Equal(t, 6.0, lp.pvMaxCurrent(-minimum, 0, true, true))
	// A previous active boost cannot use the battery to pass the restart gate.
	lp.batteryBoost = boostContinue
	require.Zero(t, lp.pvMaxCurrent(-1500, 0, true, true))
}

func TestCloudBoostSocBoundaryAndPVOnlyContinuation(t *testing.T) {
	for _, tc := range []struct{ soc, pv, want float64 }{
		{50.1, 1500, 4800}, {50, 1500, 0}, {49.9, 1500, 0},
		{50, 4500, 4500}, {49.9, 4500, 4500},
	} {
		lp, s := cloudLoadpoint()
		s.soc = tc.soc
		current := lp.pvMaxCurrent(lp.chargePower-tc.pv, 3300, true, true)
		require.InDelta(t, tc.want, grid.CurrentToPower(current, 3), 0.0001)
		if tc.soc <= 50 {
			require.Equal(t, boostHold, lp.GetBatteryBoost())
		}
	}
}

func TestCloudBoostHouseDemandAndDischargeAreNotDoubleCounted(t *testing.T) {
	for _, battery := range []float64{-2000, 0, 1000, 3300} {
		lp, _ := cloudLoadpoint()
		// PV 3kW, house 1.5kW, EV 4.8kW. Whatever part the battery
		// already supplies, EV budget must remain 1.5 + 3.3 = 4.8kW.
		lp.chargePower = 4800
		grid := 1500 + lp.chargePower - 3000 - battery
		current := lp.pvMaxCurrent(grid+battery, battery, false, false)
		require.InDelta(t, 4800.0, grid.CurrentToPower(current, 3), 0.0001)
	}
	lp, _ := cloudLoadpoint()
	// PV does not even cover the house: 1.5kW of battery capacity belongs
	// to the house first. Remaining EV budget is only 1.8kW, hence pause.
	require.Zero(t, lp.pvMaxCurrent(lp.chargePower+1500, 3300, false, false))
}

func TestCloudBoostReserveUnknownLimitsAndNoDisableDelay(t *testing.T) {
	lp, s := cloudLoadpoint()
	lp.Disable.Delay = time.Hour
	// 4.2kW total without reserve, 4.1kW after reserving 100W: pause
	// immediately, even though the configured disable timer is long.
	require.Zero(t, lp.pvMaxCurrent(lp.chargePower-900+100, 3300, true, true))
	s.residualPower = -500
	// Only 3.8kW are available physically; a negative configured residual
	// must not turn this into a fictitious 4.3kW allowance.
	require.Zero(t, lp.pvMaxCurrent(lp.chargePower-500-500, 3300, true, true))
	s.residualPower = 0
	s.maxDischargePower = nil
	require.Zero(t, lp.pvMaxCurrent(lp.chargePower-1500, 0, false, false))
	require.InDelta(t, 4500.0, grid.CurrentToPower(lp.pvMaxCurrent(lp.chargePower-4500, 0, false, false), 3), 0.0001)
	s.soc = math.NaN()
	require.Zero(t, lp.pvMaxCurrent(lp.chargePower-1500, 0, false, false))
	require.Zero(t, lp.pvMaxCurrent(math.NaN(), 0, false, false))
}

func TestCloudBoostEnableDelayAndCurrentCap(t *testing.T) {
	lp, _ := cloudLoadpoint()
	lp.enabled, lp.status, lp.chargePower = false, api.StatusB, 0
	lp.Enable.Delay = time.Minute
	require.Zero(t, lp.pvMaxCurrent(-4500, 0, false, false))
	lp.clock.(*clock.Mock).Add(time.Minute)
	require.Equal(t, 6.0, lp.pvMaxCurrent(-4500, 0, false, false))
	lp.enabled, lp.status, lp.chargePower = true, api.StatusC, 4500
	require.Equal(t, 16.0, lp.pvMaxCurrent(-20000, 0, false, false))
}
