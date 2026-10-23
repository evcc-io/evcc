package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"path/filepath"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/batterycontrol"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/hems/hems"
)

func (site *Site) updatePVBatteryControl(state siteState, valid, allowed bool) {
	site.batteryPVMu.Lock()
	defer site.batteryPVMu.Unlock()
	if site.batteryPVStopped {
		return
	}
	if site.batteryPVSessions == nil {
		site.batteryPVSessions = make(map[string]*batterycontrol.Session)
	}
	if site.batteryPVJournals == nil {
		site.batteryPVJournals = make(map[string]*batterycontrol.JournalSession)
	}
	if dimmed := hems.Dimmed(site.hems); dimmed != nil && *dimmed {
		allowed = false
	}
	if site.pvVehiclePriority(state) {
		allowed = false
		site.log.DEBUG.Println("PV allocation: returning own battery control for vehicle PV budget")
	}
	valid = valid && site.gridMeter != nil && len(site.pvMeters) > 0 && finiteBatteryPVValue(state.gridPower) && finiteBatteryPVValue(state.pvPower)
	grid, pv := state.gridPower, state.pvPower
	if valid {
		current, err := site.gridMeter.Instance().CurrentPower()
		valid = err == nil && finiteBatteryPVValue(current)
		grid = math.Max(grid, current)
		var measuredPV float64
		for _, dev := range site.pvMeters {
			power, err := dev.Instance().CurrentPower()
			valid = valid && err == nil && finiteBatteryPVValue(power)
			measuredPV += max(0, power)
		}
		pv = min(pv, measuredPV)
	}
	discharges := make(map[string]float64)
	charges := make(map[string]float64)
	var totalDischarge float64
	for _, dev := range site.batteryMeters {
		power, err := dev.Instance().CurrentPower()
		valid = valid && err == nil && finiteBatteryPVValue(power)
		discharges[dev.Config().Name] = max(0, power)
		charges[dev.Config().Name] = max(0, -power)
		totalDischarge += max(0, power)
	}
	allocated := site.pendingVehiclePower()
	if allocated > 0 {
		site.log.DEBUG.Printf("PV allocation: reserving %.0fW of unmeasured vehicle demand", allocated)
	}
	for _, dev := range site.batteryMeters {
		meter, name := dev.Instance(), dev.Config().Name
		reader, hasReader := api.Cap[api.BatteryControlStateReader](meter)
		ctrl, hasControl := api.Cap[api.BatteryPVLeaseController](meter)
		if hasControl {
			reader, hasReader = ctrl, true
		}
		if !hasReader && !api.HasCap[api.BatteryChargePowerController](meter) {
			continue
		}
		session := site.batteryPVSessions[name]
		if session == nil {
			session = new(batterycontrol.Session)
			site.batteryPVSessions[name] = session
		}
		maxSoc, maxPower := 100.0, 0.0
		if limiter, ok := api.Cap[api.BatterySocLimiter](meter); ok {
			var err error
			_, maxSoc, err = limiter.GetSocLimits()
			if err != nil {
				maxSoc = math.NaN()
			}
			if maxSoc == 0 {
				maxSoc = 100
			}
		}
		if limiter, ok := api.Cap[api.BatteryPowerLimiter](meter); ok {
			var err error
			maxPower, _, err = limiter.GetPowerLimits()
			if err != nil {
				maxPower = math.NaN()
			}
		}
		before, reason := session.Phase, session.Reason
		input := batterycontrol.Input{
			Valid: valid, Allowed: allowed, Grid: grid,
			UnavailablePower: totalDischarge - discharges[name] + allocated,
			PV:               max(0, pv-allocated), Reserve: max(100, site.GetResidualPower()),
			StartPower: site.GetBatteryPVStartPower(), MaxPower: maxPower, MaxSoc: maxSoc,
		}
		var stepErr error
		restore, canRestore := api.Cap[api.BatteryControlRestorer](meter)
		setter, canSet := api.Cap[api.BatteryChargePowerController](meter)
		if hasReader && canRestore && canSet && !hasControl {
			journal := site.batteryPVJournals[name]
			if journal == nil {
				var store batterycontrol.JournalStore
				if path := db.FilePath(); db.Instance != nil && path != "" && filepath.Base(path) != ":memory:" {
					opened, err := batterycontrol.OpenFileJournal(fmt.Sprintf("%s.pv-%x.jsonl", path, sha256.Sum256([]byte(name))))
					if err != nil {
						site.log.ERROR.Printf("PV journal: %v", err)
					} else {
						store = opened
					}
				}
				journal = batterycontrol.NewJournalSession(session, store, reader, setter, restore)
				site.batteryPVJournals[name] = journal
			}
			journal.SetDevice(reader, setter, restore)
			stepErr = journal.Step(input)
		} else {
			stepErr = session.Step(reader, ctrl, input)
		}
		if err := stepErr; err != nil {
			site.log.ERROR.Printf("active PV battery charging: %s: %v", name, err)
		}
		reserved := session.Power()
		if journal := site.batteryPVJournals[name]; journal != nil {
			reserved = journal.ReservedPower()
		}
		allocated += max(0, reserved-charges[name])
		if before != session.Phase || reason != session.Reason {
			site.log.INFO.Printf("active PV battery charging: battery=%s state=%s reason=%s", name, session.Phase, session.Reason)
		}
		if journal := site.batteryPVJournals[name]; journal != nil {
			site.log.DEBUG.Printf("active PV battery charging: battery=%s grid=%.0fW pv=%.0fW controllerTarget=%.0fW lastChargeRequest=%.0fW state=%s lastObservedMode=%s lastObservedNativeMode=%s lastObservedPower=%.0fW nativeCharging=%t observationValid=%t observedAt=%s journalStage=%s", name, grid, pv, session.Power(), journal.Record.Power, session.Phase, session.ObservedMode, session.ObservedNativeMode, session.ObservedPower, session.ObservedNativeCharging, journal.ObservationValid, journal.ObservationAt.Format(time.RFC3339Nano), journal.Record.Stage)
		} else {
			site.log.DEBUG.Printf("active PV battery charging: battery=%s grid=%.0fW pv=%.0fW target=%.0fW state=%s actualMode=%s nativeMode=%s actualPower=%.0fW", name, grid, pv, session.Power(), session.Phase, session.ObservedMode, session.ObservedNativeMode, session.ObservedPower)
		}
	}
	site.publish("batteryPVControl", site.pvBatteryStatus())
}

func (site *Site) stopPVBatteryControl() {
	site.batteryPVMu.Lock()
	defer site.batteryPVMu.Unlock()
	site.batteryPVStopped = true
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, dev := range site.batteryMeters {
		name := dev.Config().Name
		if journal := site.batteryPVJournals[name]; journal != nil {
			err := journal.Shutdown(ctx)
			if err != nil {
				site.log.ERROR.Printf("PV battery journal shutdown %s: %v", name, err)
			}
			if err := journal.Close(); err != nil {
				site.log.ERROR.Printf("PV journal close %s: %v", name, err)
			}
			continue
		}
		if session := site.batteryPVSessions[name]; session != nil {
			ctrl, _ := api.Cap[api.BatteryPVLeaseController](dev.Instance())
			if err := session.Stop(ctrl, "evcc shutdown"); err != nil {
				site.log.ERROR.Printf("active PV battery charging: %s shutdown release: %v; device lease must expire", name, err)
			}
		}
	}
}

// Caller holds batteryPVMu.
func (site *Site) pvBatteryStatus() map[string]map[string]any {
	status := make(map[string]map[string]any)
	for name, session := range site.batteryPVSessions {
		entry := map[string]any{"state": session.Phase, "reason": session.Reason, "power": session.Power(), "actualMode": session.ObservedMode, "nativeMode": session.ObservedNativeMode, "actualPower": session.ObservedPower}
		if j := site.batteryPVJournals[name]; j != nil {
			entry["revision"] = j.Record.Revision
			entry["updated"] = j.Record.Updated
			entry["expectedPower"] = j.Record.Power
			entry["previousMode"] = j.Record.PreviousMode
			entry["observationValid"] = j.ObservationValid
			entry["observedAt"] = j.ObservationAt
			entry["journalStage"] = j.Record.Stage
		}
		status[name] = entry
	}
	return status
}

func (site *Site) ResolveBatteryPVControl(name, action, revision string) error {
	site.batteryPVMu.Lock()
	defer site.batteryPVMu.Unlock()
	if site.batteryPVStopped {
		return fmt.Errorf("site shutting down")
	}
	journal := site.batteryPVJournals[name]
	if journal == nil {
		return fmt.Errorf("battery journal not found")
	}
	if err := journal.Resolve(action, revision); err != nil {
		return err
	}
	site.publish("batteryPVControl", site.pvBatteryStatus())
	return nil
}
