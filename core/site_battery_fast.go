package core

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
)

// The battery fast loop is the complete battery power controller. The main loop
// (buildBatterySnapshot) only decides participation - it sets the battery mode and
// publishes a snapshot (per-battery SoC/limits/caps + config) once per cycle. Every
// power decision - direction, tiering, selection, swaps, scaling, stops - is made here,
// every tick, off fresh grid and battery readings. Direction is continuous (the sign of
// the energy-balance need), so there is no separate charge/discharge decision to wait on.
const (
	batteryFastLoopPeriod = 1 * time.Second  // matched to the (DSMR P1) grid telegram cadence
	batterySnapshotMaxAge = 30 * time.Second // ignore snapshots when the main loop stalled
	fastLoopSkewThreshold = 100.0            // W; meter consistency guard (see batteryFastTick)

	// direction anti-chatter: a genuine charge<->discharge reversal must persist this long,
	// and repeated reversals near the crossing back off (doubling up to max, reset when calm)
	fastLoopFlipDwell      = 1 * time.Second
	fastLoopFlipBackoffMin = 2 * time.Second
	fastLoopFlipBackoffMax = 60 * time.Second
	fastLoopFlipCalm       = 120 * time.Second

	socSwitchThreshold = 3.0  // % SoC divergence before swapping a sticky battery
	minEffectiveShare  = 50.0 // W; below this per-battery share, concentrate (no power limiter)
)

type batteryPlanDirection int

const (
	batteryPlanIdle batteryPlanDirection = iota
	batteryPlanCharge
	batteryPlanDischarge
)

func batteryPlanDirectionString(d batteryPlanDirection) string {
	switch d {
	case batteryPlanCharge:
		return "charge"
	case batteryPlanDischarge:
		return "discharge"
	default:
		return "idle"
	}
}

// batterySnapEntry is a per-battery slice of the snapshot: control handle, meter, and the
// slow-moving state (SoC, limits, caps) the main loop read this cycle.
type batterySnapEntry struct {
	ctrl         api.BatteryPowerController
	meter        api.Meter
	name         string
	soc          float64
	socOK        bool
	hasSocLimit  bool
	minSoc       float64
	maxSoc       float64
	chargeCap    float64 // W; 0 = uncapped
	dischargeCap float64 // W; 0 = uncapped
}

// batterySnapshot is the main loop -> fast loop contract, rebuilt each main cycle under
// batteryPlanMu. It carries no power or direction - the fast loop derives those.
type batterySnapshot struct {
	enabled             bool // fast loop may drive power (Hold mode, solar control, not overridden)
	pool                bool
	tiering             bool
	sticky              bool
	tapering            bool
	calibration         bool
	chargeOffset        float64 // residual, or 0 below prioritySoc
	dischargeOffset     float64 // residual
	dischargeEvExcluded float64 // EV power the battery must not cover (discharge only)
	threshold           float64 // dead band
	created             time.Time
	batteries           []batterySnapEntry
}

// fastEntry references a snapshot battery for the per-tick selection working set.
type fastEntry struct {
	*batterySnapEntry
}

// fastEntriesAll wraps every snapshot battery as a fastEntry (used for stop-all paths).
func fastEntriesAll(snap *batterySnapshot) []fastEntry {
	all := make([]fastEntry, len(snap.batteries))
	for i := range snap.batteries {
		all[i] = fastEntry{batterySnapEntry: &snap.batteries[i]}
	}
	return all
}

// batteryFastLoop runs the controller tick until stopC closes.
func (site *Site) batteryFastLoop(stopC chan struct{}) {
	if site.gridMeter == nil {
		return
	}
	ticker := time.NewTicker(batteryFastLoopPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-stopC:
			return
		case <-ticker.C:
			site.batteryFastTick()
		}
	}
}

func (site *Site) batteryFastTick() {
	site.batteryPlanMu.Lock()
	defer site.batteryPlanMu.Unlock()

	snap := site.batterySnapshot
	if snap == nil {
		batteryLog.TRACE.Print("solar power (fast): parked (no snapshot)")
		return
	}
	if !snap.enabled || len(snap.batteries) == 0 || time.Since(snap.created) > batterySnapshotMaxAge {
		batteryLog.TRACE.Printf("solar power (fast): parked (enabled=%v, %d batteries, age %s)",
			snap.enabled, len(snap.batteries), time.Since(snap.created).Round(time.Second))
		return
	}

	if site.batteryStopped == nil {
		site.batteryStopped = make(map[string]int)
	}

	gridPower, err := site.gridMeter.CurrentPower()
	if err != nil {
		batteryLog.DEBUG.Printf("solar power (fast): grid power: %v", err)
		return
	}

	// meter guard rule 1 - stale grid: an unchanged reading carries no new information and
	// pairing it with a fresher battery reading would double-count the battery's ramp
	firstTick := !site.batteryGuardValid
	if !firstTick && gridPower == site.batteryLastGrid {
		batteryLog.TRACE.Printf("solar power (fast): stale grid %.0fW, skip", gridPower)
		return
	}

	// measured battery power of all snapshot batteries, read in parallel
	powers := make([]float64, len(snap.batteries))
	errs := make([]error, len(snap.batteries))
	var rwg sync.WaitGroup
	for i := range snap.batteries {
		rwg.Add(1)
		go func() {
			defer rwg.Done()
			powers[i], errs[i] = snap.batteries[i].meter.CurrentPower()
		}()
	}
	rwg.Wait()

	var battPower float64
	for i := range snap.batteries {
		if errs[i] != nil {
			batteryLog.DEBUG.Printf("solar power (fast): %s power: %v", snap.batteries[i].name, errs[i])
			return
		}
		battPower += powers[i]
	}

	// meter guard rule 2 - sampling skew: with constant load Δgrid + Δbatt ≈ 0; when the
	// battery moved without the grid reflecting it, the registers are out of sync - skip
	dGrid, dBatt := gridPower-site.batteryLastGrid, battPower-site.batteryLastBatt
	site.batteryLastGrid, site.batteryLastBatt, site.batteryGuardValid = gridPower, battPower, true
	if !firstTick && math.Abs(dBatt) > fastLoopSkewThreshold && math.Abs(dGrid+dBatt) > fastLoopSkewThreshold {
		batteryLog.TRACE.Printf("solar power (fast): meters inconsistent (Δgrid %.0fW, Δbattery %.0fW), skip", dGrid, dBatt)
		return
	}

	// Energy-balance targets (battery convention: positive = discharging). Ramp-invariant
	// because measured battPower is added back. Only one is positive away from the crossing.
	dischargeTarget := battPower + gridPower + snap.dischargeOffset - snap.dischargeEvExcluded
	chargeTarget := -battPower - (gridPower + snap.chargeOffset)

	// desired direction from the fresh signal
	desired := batteryPlanIdle
	switch {
	case dischargeTarget > snap.threshold && dischargeTarget >= chargeTarget:
		desired = batteryPlanDischarge
	case chargeTarget > snap.threshold:
		desired = batteryPlanCharge
	}

	batteryLog.TRACE.Printf("solar power (fast): grid=%.0fW batt=%.0fW dis=%.0fW chg=%.0fW committed=%s desired=%s",
		gridPower, battPower, dischargeTarget, chargeTarget,
		batteryPlanDirectionString(site.batteryFastDirection), batteryPlanDirectionString(desired))

	// direction arbitration with anti-chatter: a reversal must persist for the dwell and is
	// rate-limited by an adaptive backoff. Same-direction changes pass through immediately.
	direction := site.batteryArbitrateDirection(desired)

	switch direction {
	case batteryPlanCharge:
		// clamp ≥ 0: while a flip is pending the held direction's target is negative, so it
		// simply ramps to zero rather than commanding negative power
		site.fastControl(snap, batteryPlanCharge, math.Max(0, chargeTarget))
	case batteryPlanDischarge:
		site.fastControl(snap, batteryPlanDischarge, math.Max(0, dischargeTarget))
	default:
		site.stopBatteries(fastEntriesAll(snap))
		site.batteryChargeActive, site.batteryDischargeActive = nil, nil
		site.batteryChargeTier, site.batteryDischargeTier = 0, 0
	}
	site.batteryFastDirection = direction
}

// batteryArbitrateDirection returns the direction to drive this tick. A reversal from the
// committed direction must persist for fastLoopFlipDwell and is spaced by an adaptive
// backoff (doubling up to max on repeated reversals, reset after a calm gap). While a
// reversal is pending the committed direction is held (its target has clamped to ~0, so it
// simply ramps down). Idle and same-direction pass through immediately.
func (site *Site) batteryArbitrateDirection(desired batteryPlanDirection) batteryPlanDirection {
	committed := site.batteryFastDirection

	if desired == committed || desired == batteryPlanIdle || committed == batteryPlanIdle {
		site.batteryFlipSince = time.Time{}
		return desired
	}

	// desired is the opposite active direction: gate it
	if site.batteryFlipSince.IsZero() {
		site.batteryFlipSince = time.Now()
		return committed
	}
	if time.Since(site.batteryFlipSince) < fastLoopFlipDwell {
		return committed
	}
	gap := time.Since(site.lastBatteryFlipRequest)
	if gap < site.batteryFlipBackoff {
		return committed
	}
	if gap < fastLoopFlipCalm {
		site.batteryFlipBackoff = min(2*site.batteryFlipBackoff, fastLoopFlipBackoffMax)
		if site.batteryFlipBackoff < fastLoopFlipBackoffMin {
			site.batteryFlipBackoff = fastLoopFlipBackoffMin
		}
	} else {
		site.batteryFlipBackoff = fastLoopFlipBackoffMin
	}
	site.batteryFlipSince = time.Time{}
	site.lastBatteryFlipRequest = time.Now()
	batteryLog.DEBUG.Printf("solar power (fast): direction flip %s → %s (next flip ≥ %s)",
		batteryPlanDirectionString(committed), batteryPlanDirectionString(desired), site.batteryFlipBackoff)
	return desired
}

// fastControl runs the full selection/tiering/scaling for one direction and commands the
// batteries. target is the signed magnitude to deliver in that direction (positive).
func (site *Site) fastControl(snap *batterySnapshot, dir batteryPlanDirection, target float64) {
	charge := dir == batteryPlanCharge

	// eligibility filter: charge drops units at maxSoc (unless calibrating); discharge drops
	// units at/below minSoc and fails closed on an unreadable SoC (never over-drain)
	var active, ineligible []fastEntry
	for i := range snap.batteries {
		e := &snap.batteries[i]
		fe := fastEntry{batterySnapEntry: e}
		if charge {
			if !snap.calibration && e.hasSocLimit && e.maxSoc > 0 && e.socOK && e.soc >= e.maxSoc {
				ineligible = append(ineligible, fe)
				continue
			}
		} else {
			if !e.socOK {
				ineligible = append(ineligible, fe)
				continue
			}
			if e.hasSocLimit && e.soc <= e.minSoc {
				ineligible = append(ineligible, fe)
				continue
			}
		}
		active = append(active, fe)
	}

	var deferStop []fastEntry
	deferStop = append(deferStop, ineligible...)
	if len(active) == 0 {
		site.stopBatteries(fastEntriesAll(snap))
		return
	}

	capOf := func(e fastEntry) float64 {
		if charge {
			return e.chargeCap
		}
		return e.dischargeCap
	}

	// selection: tiering + sticky, unless pool mode uses all active equally
	if !snap.pool {
		var maxPerBat float64
		for _, e := range active {
			if c := capOf(e); c > 0 && (maxPerBat == 0 || c < maxPerBat) {
				maxPerBat = c
			}
		}

		if maxPerBat > 0 && snap.tiering {
			tierPerBat := maxPerBat * batteryTierFraction
			tier := &site.batteryChargeTier
			activeNames := &site.batteryChargeActive
			if !charge {
				tier = &site.batteryDischargeTier
				activeNames = &site.batteryDischargeActive
			}
			*tier = computeTier(target, tierPerBat, *tier, len(active))
			needed := *tier

			if needed < len(active) {
				sel, stop := site.selectSticky(active, needed, charge, activeNames)
				deferStop = append(deferStop, stop...)
				active = sel
			} else {
				*activeNames = nil
			}
		} else if maxPerBat == 0 {
			// no power limiter: concentrate on the single best unit if the share is too small
			share := target / float64(len(active))
			if share < minEffectiveShare && len(active) > 1 {
				best := 0
				for i, e := range active {
					if charge {
						if e.socOK && (i == 0 || e.soc < active[best].soc) {
							best = i
						}
					} else if e.socOK && (i == 0 || e.soc > active[best].soc) {
						best = i
					}
				}
				for i, e := range active {
					if i != best {
						deferStop = append(deferStop, e)
					}
				}
				active = active[best : best+1]
			}
		}
	}

	// scale and command (parallel writes), applying per-battery cap and charge taper
	share := target / float64(len(active))
	commands := make([]float64, len(active))
	for i, e := range active {
		p := share
		if c := capOf(e); c > 0 && p > c {
			p = c
		}
		if charge && snap.tapering && !snap.calibration && e.hasSocLimit && e.maxSoc > 0 && e.socOK && e.soc > e.maxSoc-chargeTaperRange {
			factor := (e.maxSoc - e.soc) / chargeTaperRange
			if factor < chargeMinFactor {
				factor = chargeMinFactor
			}
			p *= factor
		}
		commands[i] = p
	}
	site.sendBatteries(active, charge, commands)

	var total float64
	for _, c := range commands {
		total += c
	}
	batteryLog.DEBUG.Printf("solar power (fast): %s %.0fW across %d/%d batteries (grid target %.0fW)",
		batteryPlanDirectionString(dir), total, len(active), len(snap.batteries), target)

	site.stopBatteries(deferStop)
}

// selectSticky keeps the current active set stable, swapping one unit only when a candidate
// is more than socSwitchThreshold better. charge fills lowest-SoC first, discharge drains
// highest-SoC first. Returns the selected set and the units to stop (the leftover candidates).
func (site *Site) selectSticky(active []fastEntry, needed int, charge bool, activeNames *[]string) (sel, cand []fastEntry) {
	prev := make(map[string]bool, len(*activeNames))
	for _, n := range *activeNames {
		prev[n] = true
	}
	for _, e := range active {
		if prev[e.name] {
			sel = append(sel, e)
		} else {
			cand = append(cand, e)
		}
	}

	better := func(a, b fastEntry) bool { // a strictly preferred over b
		if charge {
			return a.soc < b.soc // lower SoC first
		}
		return a.soc > b.soc // higher SoC first
	}

	if len(sel) != needed {
		// stored set no longer valid: reselect the N best by SoC
		all := append([]fastEntry{}, active...)
		sort.Slice(all, func(i, j int) bool { return better(all[i], all[j]) })
		sel = all[:needed]
		cand = all[needed:]
	} else {
		// find the worst in sel and swap in a clearly-better candidate
		worst := 0
		for i, e := range sel {
			if i == 0 || better(sel[worst], e) {
				worst = i
			}
		}
		for ci, c := range cand {
			// swap when a candidate is clearly better than our worst pick:
			// charge → candidate emptier (lower SoC); discharge → candidate fuller (higher SoC)
			diff := sel[worst].soc - c.soc
			if !charge {
				diff = c.soc - sel[worst].soc
			}
			if c.socOK && sel[worst].socOK && diff > socSwitchThreshold {
				batteryLog.DEBUG.Printf("solar power (fast): %s swap %s (%.0f%%) → %s (%.0f%%)",
					map[bool]string{true: "charge", false: "discharge"}[charge],
					sel[worst].name, sel[worst].soc, c.name, c.soc)
				sel[worst], cand[ci] = c, sel[worst]
				break
			}
		}
	}

	*activeNames = make([]string, len(sel))
	for i, e := range sel {
		(*activeNames)[i] = e.name
	}
	return sel, cand
}

// sendBatteries writes power commands in parallel (each battery has its own connection) and
// clears the stop bookkeeping for the commanded units.
func (site *Site) sendBatteries(active []fastEntry, charge bool, commands []float64) {
	var wg sync.WaitGroup
	for i, e := range active {
		delete(site.batteryStopped, e.name)
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			if charge {
				err = e.ctrl.SetBatteryChargePower(commands[i])
			} else {
				err = e.ctrl.SetBatteryDischargePower(commands[i])
			}
			if err != nil {
				batteryLog.ERROR.Printf("solar power (fast): %s: %v", e.name, err)
			}
		}()
	}
	wg.Wait()
}

// stopBatteries commands 0 to the given batteries, skipping units already stopped and
// re-sending only every stopRefreshTicks as an RS485 watchdog heartbeat.
func (site *Site) stopBatteries(entries []fastEntry) {
	for _, e := range entries {
		if n, ok := site.batteryStopped[e.name]; ok && n < stopRefreshTicks {
			site.batteryStopped[e.name] = n + 1
			continue
		}
		failed := false
		if err := e.ctrl.SetBatteryChargePower(0); err != nil {
			batteryLog.ERROR.Printf("battery charge power: %v", err)
			failed = true
		}
		if err := e.ctrl.SetBatteryDischargePower(0); err != nil {
			batteryLog.ERROR.Printf("battery discharge power: %v", err)
			failed = true
		}
		if failed {
			delete(site.batteryStopped, e.name)
		} else {
			site.batteryStopped[e.name] = 0
		}
	}
}
