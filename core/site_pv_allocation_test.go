package core

import (
	"path/filepath"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/batterycontrol"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/require"
)

func TestPVAllocationReservesUnmeasuredVehicleDemand(t *testing.T) {
	b := &leasedPVBattery{observedPVBattery: observedPVBattery{state: api.BatteryControlState{Mode: api.BatteryOperatingAuto, Soc: 57}}}
	s := guardedSite(&b.observedPVBattery)
	s.batteryMeters = []config.Device[api.Meter]{config.NewStaticDevice[api.Meter](config.Named{Name: "battery"}, b)}
	lp, _ := cloudLoadpoint()
	lp.offeredCurrent, lp.chargePower = 8.11, 4140
	s.loadpoints = []*Loadpoint{lp}
	s.gridMeter.Instance().(*observedPVBattery).state.Power = -1551
	s.pvMeters[0].Instance().(*observedPVBattery).state.Power = 5982
	require.InDelta(t, 1455.9, s.pendingVehiclePower(), 0.01)
	s.updatePVBatteryControl(siteState{gridPower: -1551, pvPower: 5982}, true, true)
	require.Zero(t, b.target) // the same 1.45 kW cannot be handed to the battery
	// At the EV current cap, actual remaining surplus may charge the battery.
	lp.offeredCurrent, lp.chargePower = 16, 11040
	require.Zero(t, s.pendingVehiclePower())
	s.updatePVBatteryControl(siteState{gridPower: -1551, pvPower: 15000}, true, true)
	require.Equal(t, 1451.0, b.target)
}

func TestPVAllocationHandsOwnedChargingToWaitingVehicle(t *testing.T) {
	b := &journalSiteBattery{observedPVBattery{state: api.BatteryControlState{Identity: "device", Mode: api.BatteryOperatingAuto, NativeMode: "2", Soc: 57}}}
	s := guardedSite(&b.observedPVBattery)
	s.bufferSoc = 50
	s.batteryMeters = []config.Device[api.Meter]{config.NewStaticDevice[api.Meter](config.Named{Name: "battery"}, b)}
	store, err := batterycontrol.OpenFileJournal(filepath.Join(t.TempDir(), "journal.jsonl"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	session := new(batterycontrol.Session)
	j := batterycontrol.NewJournalSession(session, store, b, b, b)
	s.batteryPVJournals = map[string]*batterycontrol.JournalSession{"battery": j}
	s.batteryPVSessions = map[string]*batterycontrol.Session{"battery": session}
	grid := s.gridMeter.Instance().(*observedPVBattery)
	pv := s.pvMeters[0].Instance().(*observedPVBattery)
	grid.state.Power, pv.state.Power = -3400, 5000
	state := siteState{gridPower: -3400, pvPower: 5000}
	state.battery.Soc = 57
	s.updatePVBatteryControl(state, true, true)
	s.updatePVBatteryControl(state, true, true)
	require.Equal(t, batterycontrol.Active, session.Phase)
	require.Equal(t, -3300.0, b.state.Power)
	lp, _ := cloudLoadpoint()
	lp.site, lp.mode, lp.batteryBoost = s, api.ModeSmart, boostDisabled
	lp.status, lp.enabled, lp.chargePower = api.StatusB, false, 0
	s.loadpoints = []*Loadpoint{lp}
	grid.state.Power, state.gridPower = -1100, -1100
	state.battery.Power, state.batteryFixedChargePower = -3300, 3300
	res := s.sitePower(state, 0, 0)
	require.Zero(t, lp.pvMaxCurrent(res.power, -3300, true, true))
	s.updatePVBatteryControl(state, true, true)
	require.Equal(t, batterycontrol.Recovering, session.Phase)
	grid.state.Power, state.gridPower = -4400, -4400
	state.battery.Power, state.batteryFixedChargePower = 0, 0
	s.updatePVBatteryControl(state, true, true)
	require.Equal(t, batterycontrol.Observing, session.Phase)
	require.Equal(t, "idle", j.Record.Stage)
	res = s.sitePower(state, 0, 0)
	require.Equal(t, 6.0, lp.pvMaxCurrent(res.power, 0, true, true))
	lp.enabled, lp.offeredCurrent = true, 6
	s.updatePVBatteryControl(state, true, true)
	require.Equal(t, "idle", j.Record.Stage)
	require.Equal(t, []api.BatteryMode{api.BatteryCharge, api.BatteryNormal}, b.writes)
}

func TestPVAllocationDoesNotSpendUnmeasuredReduction(t *testing.T) {
	lp, _ := cloudLoadpoint()
	lp.offeredCurrent, lp.chargePower = 6, 5500
	s := NewSite()
	s.loadpoints = []*Loadpoint{lp}
	require.Zero(t, s.pendingVehiclePower())
	lp.offeredCurrent, lp.chargePower = 8, 4000
	other, _ := cloudLoadpoint()
	other.offeredCurrent, other.chargePower = 6, 0
	s.loadpoints = append(s.loadpoints, other)
	require.InDelta(t, 5660, s.pendingVehiclePower(), 0.01)
	require.InDelta(t, 4140, s.pendingVehiclePowerExcept(lp), 0.01)
	other.enabled = false
	require.InDelta(t, 1520, s.pendingVehiclePower(), 0.01)
}

func TestPVVehiclePriorityRequiresEnoughPVAndOwnJournal(t *testing.T) {
	b := &journalSiteBattery{observedPVBattery{state: api.BatteryControlState{Mode: api.BatteryOperatingManual, Power: -3300, Soc: 57}}}
	s := guardedSite(&b.observedPVBattery)
	s.batteryMeters = []config.Device[api.Meter]{config.NewStaticDevice[api.Meter](config.Named{Name: "battery"}, b)}
	lp, _ := cloudLoadpoint()
	lp.site, lp.batteryBoost, lp.mode = s, boostDisabled, api.ModeSmart
	lp.enabled, lp.status, lp.chargePower = false, api.StatusB, 0
	s.loadpoints = []*Loadpoint{lp}
	state := siteState{gridPower: -1000}
	state.battery.Soc = 57
	// Foreign manual charging is not reclaimable, even though its physical power is known.
	require.False(t, s.pvVehiclePriority(state))
	s.batteryPVJournals = map[string]*batterycontrol.JournalSession{
		"battery": {Record: batterycontrol.Record{Stage: "active"}},
	}
	require.True(t, s.pvVehiclePriority(state)) // 1000 + 3300 - 100 >= 4140
	state.gridPower = -500
	require.False(t, s.pvVehiclePriority(state))
	state.gridPower = -1000
	s.prioritySoc = 80
	require.False(t, s.pvVehiclePriority(state))
	s.prioritySoc = 0
	lp.status, lp.enabled, lp.offeredCurrent = api.StatusC, true, 16
	require.False(t, s.pvVehiclePriority(state)) // EV capped: battery may take remainder
	lp.offeredCurrent = 8
	require.True(t, s.pvVehiclePriority(state))
}

func TestPVAllocationReservesPendingBatteryCommands(t *testing.T) {
	b := &journalSiteBattery{observedPVBattery{state: api.BatteryControlState{Power: -200}}}
	s := guardedSite(&b.observedPVBattery)
	s.batteryMeters = []config.Device[api.Meter]{config.NewStaticDevice[api.Meter](config.Named{Name: "battery"}, b)}
	for _, stage := range []string{"intent", "active", "releasing"} {
		s.batteryPVJournals = map[string]*batterycontrol.JournalSession{
			"battery": {Record: batterycontrol.Record{Stage: stage, Power: 1451}},
		}
		require.Equal(t, 1251.0, s.pendingBatteryPower())
	}
	b.state.Power = -1500
	require.Zero(t, s.pendingBatteryPower()) // no credit for a reduction not yet measured
	s.batteryPVJournals["battery"].Record.Stage = "idle"
	b.state.Power = 0
	require.Zero(t, s.pendingBatteryPower())
}
