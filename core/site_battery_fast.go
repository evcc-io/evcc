package core

import (
	"math"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
)

// The battery fast loop is a thin companion to applyBatterySolarPower. All decisions
// (charge/discharge direction, tiering, sticky selection, swaps, stops, mode handling)
// remain in the main loop. The fast loop owns the power values of the currently active
// batteries: it re-scales them against fresh grid readings, closing the gap between
// main loop ticks. It never changes direction, never selects batteries and never
// sends stop commands - when its correction reaches zero it clamps and waits for
// the main loop to decide.
const (
	batteryFastLoopPeriod = 1 * time.Second // matched to the (DSMR P1) grid telegram cadence;
	// ticking faster than the meter refreshes only re-chews stale samples - every tick now reads
	// a fresh telegram, which also removes the stale-read overshoot that gain 1.0 used to amplify
	batteryPlanMaxAge = 30 * time.Second // ignore plans when the main loop stopped updating them
	fastLoopMinDelta  = 10.0             // W; skip Modbus writes below the grid meter noise floor
	fastLoopGain      = 1.0              // full one-tick correction toward the measured target.
	// Kept aggressive for reactivity; the target is absolute (no integration) and gross sampling
	// skew is handled by the consistency guard below. If near-zero/phase-imbalance ringing returns,
	// prefer a near-zero deadband (raise fastLoopMinDelta) over lowering the gain
	fastLoopSkewThreshold = 100.0            // W; see meter consistency guard in batteryFastTick
	fastLoopHeartbeat     = 10 * time.Second // re-send the current setpoints when no write
	// happened for this long, keeping the inverters' RS485 watchdog alive now that the
	// main loop no longer re-commands active batteries every tick
	fastLoopTierMargin = 50.0 // W of unmet demand beyond engaged capacity before the
	// fast loop engages another battery (Marstek minimum effective power)
	fastLoopShortfall     = 150.0 // W; engaged set delivering at least this far below the
	// commanded total counts as under-delivery (a unit ACKs but can't physically deliver)
	fastLoopShortfallDwell = 4 // consecutive under-delivering ticks (≈4s at 1s tick, longer
	// than the inverter ramp) before engaging a standby, so normal ramp-up doesn't trip it
	fastLoopFlipDwell = 2 * time.Second // the opposite direction must stay wanted (past the
	// dead band) this long before poking the main loop to re-decide direction. Wall-clock,
	// not tick count, so stale-grid ticks that don't refresh the reading do not stretch it.
	// The opposite need is ramp-invariant, so this runs during the ramp - no wait for zero
	fastLoopFlipCooldown = 15 * time.Second // minimum spacing between fast-loop flip pokes,
	// bounding charge<->discharge thrash when power hovers near the crossing. The main loop's
	// scheduled tick can still flip during the cooldown; this only rate-limits the fast path

)

type batteryPlanDirection int

const (
	batteryPlanIdle batteryPlanDirection = iota
	batteryPlanCharge
	batteryPlanDischarge
)

type batteryPlanEntry struct {
	ctrl  api.BatteryPowerController
	meter api.Meter
	name  string
	cap   float64 // effective per-battery power limit in W incl. taper; 0 = uncapped
}

// batteryControlPlan is the contract between the main loop and the fast loop.
// The main loop replaces it on every tick; the fast loop adjusts total between ticks.
// Both access it under batteryPlanMu.
type batteryControlPlan struct {
	direction batteryPlanDirection
	entries   []batteryPlanEntry
	standby   []batteryPlanEntry // eligible batteries beyond the current tier, in engage
	// order; the fast loop turns them on (tier-up) when the engaged set saturates
	evExcluded float64 // W of EV charge power the battery must not cover (discharge only)
	gridOffset float64 // grid setpoint offset the main loop steered toward (residualPower,
	// or 0 below prioritySoc where the energy-balance formula ignores it)

	// opposite-direction parameters + shared dead band, so the fast loop can detect a
	// charge<->discharge crossing and request a main-loop re-decision (never flips itself)
	threshold          float64 // standbyPower + batteryControlDeadBand (same as the main loop)
	oppositeGridOffset float64
	oppositeEvExcluded float64
	flipSince          time.Time // when the opposite direction first became wanted (zero = not)

	total     float64 // currently commanded total power across entries
	created   time.Time
	lastWrite time.Time // last time power commands were sent (heartbeat reference)

	// previous fast tick readings for the meter consistency guard
	lastGrid, lastBatt float64
	lastValid          bool

	// consecutive ticks the engaged set delivered materially less than commanded
	// (ACKed but under-delivering); drives tier-up onto a standby battery
	shortfallTicks int
}

// batteryFastLoop runs the correction ticker until stopC closes.
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

	plan := site.batteryPlan
	if plan == nil || plan.direction == batteryPlanIdle || len(plan.entries) == 0 ||
		time.Since(plan.created) > batteryPlanMaxAge {
		return
	}

	gridPower, err := site.gridMeter.CurrentPower()
	if err != nil {
		// skip tick, next attempt in one period
		site.log.DEBUG.Printf("solar power (fast): grid power: %v", err)
		return
	}

	// Meter consistency guard rule 1 - stale grid register: the grid meter refreshes
	// its registers slower than the fast loop ticks. An identical reading carries no
	// new information; pairing it with a fresher battery reading would double-count
	// the battery's ramping contribution. Checked before the battery reads so stale
	// ticks cost a single TCP read.
	firstTick := !plan.lastValid
	if !firstTick && gridPower == plan.lastGrid {
		site.batteryFastHeartbeat(plan)
		return
	}

	// Measured battery power of the active set. Using measurements instead of the
	// commanded total is essential: during inverter ramps the commanded value is not
	// yet delivered, and integrating the still-visible grid error against it causes
	// runaway oscillation. The energy-balance target below is ramp-state invariant.
	// Reads run in parallel (each battery has its own connection) so a multi-battery
	// tier does not serialize round-trips into the tick latency.
	powers := make([]float64, len(plan.entries))
	errs := make([]error, len(plan.entries))
	var rwg sync.WaitGroup
	for i, e := range plan.entries {
		rwg.Add(1)
		go func() {
			defer rwg.Done()
			powers[i], errs[i] = e.meter.CurrentPower()
		}()
	}
	rwg.Wait()

	var battPower float64
	for i, e := range plan.entries {
		if errs[i] != nil {
			// any failed read: skip the tick, retry next period
			site.log.DEBUG.Printf("solar power (fast): %s power: %v", e.name, errs[i])
			return
		}
		battPower += powers[i]
	}

	// Meter consistency guard rule 2 - sampling skew: with constant load,
	// Δgrid + Δbattery ≈ 0 between ticks. When the battery reading moved substantially
	// without the grid reflecting it, the registers are out of sync - skip and let
	// them align. Genuine load steps (Δbattery ≈ 0) are never skipped.
	dGrid, dBatt := gridPower-plan.lastGrid, battPower-plan.lastBatt
	plan.lastGrid, plan.lastBatt, plan.lastValid = gridPower, battPower, true
	if !firstTick && math.Abs(dBatt) > fastLoopSkewThreshold && math.Abs(dGrid+dBatt) > fastLoopSkewThreshold {
		site.log.DEBUG.Printf("solar power (fast): meters inconsistent (Δgrid %.0fW, Δbattery %.0fW), skipping tick", dGrid, dBatt)
		return
	}

	// Absolute energy-balance target, same construction as the main loop's setpoint
	// (battery power convention: positive = discharging, negative = charging).
	// Steady state (grid ≈ -gridOffset) reproduces the currently delivered power, so
	// the loop is quiescent until grid moves.
	var target float64
	switch plan.direction {
	case batteryPlanDischarge:
		target = battPower + gridPower + plan.gridOffset - plan.evExcluded
	case batteryPlanCharge:
		target = -battPower - (gridPower + plan.gridOffset)
	}
	target = math.Max(plan.total+fastLoopGain*(target-plan.total), 0)

	// Tier-up: when the engaged set is saturated (target exceeds its total capacity)
	// and an eligible standby battery is available, engage the next one. The main loop
	// owns tier-down via computeTier hysteresis, so the fast loop only ever expands -
	// no flapping. Selection and order stay in the main loop (the standby list).
	engaged := site.batteryFastTierUp(plan, target, battPower)

	// Direction crossing: when the active direction has clamped to zero and the opposite
	// direction is genuinely wanted, poke the main loop to re-decide direction. The fast
	// loop never flips itself - it only shortens when the existing main-loop decision runs.
	site.batteryFastFlipCheck(plan, gridPower, battPower, target)

	if !engaged && math.Abs(target-plan.total) < fastLoopMinDelta {
		site.batteryFastHeartbeat(plan)
		return
	}

	commanded := site.batteryFastSend(plan, target)

	dir := map[batteryPlanDirection]string{batteryPlanCharge: "charge", batteryPlanDischarge: "discharge"}[plan.direction]
	site.log.DEBUG.Printf("solar power (fast): %s %.0fW → %.0fW (grid %.0fW, battery %.0fW)", dir, plan.total, commanded, gridPower, battPower)

	plan.total = commanded
}

// batteryFastHeartbeat re-sends the current setpoints when no power command was
// written for fastLoopHeartbeat, keeping the inverters' RS485 watchdog alive.
// Called from skip paths; caller holds batteryPlanMu.
func (site *Site) batteryFastHeartbeat(plan *batteryControlPlan) {
	if time.Since(plan.lastWrite) < fastLoopHeartbeat {
		return
	}
	site.batteryFastSend(plan, plan.total)
	site.log.DEBUG.Printf("solar power (fast): heartbeat %.0fW", plan.total)
}

// batteryFastTierUp engages the next standby battery when the engaged set is saturated.
// Returns true if a battery was engaged. Caller holds batteryPlanMu. The shared tier and
// active-name state is updated so the next main tick takes ownership coherently; the main
// loop remains the sole authority for tier-down and selection.
func (site *Site) batteryFastTierUp(plan *batteryControlPlan, target, measured float64) bool {
	if len(plan.standby) == 0 {
		plan.shortfallTicks = 0
		return false
	}

	// saturation is only well-defined when every engaged battery has a known cap
	var sumCaps float64
	for _, e := range plan.entries {
		if e.cap <= 0 {
			plan.shortfallTicks = 0
			return false
		}
		sumCaps += e.cap
	}

	// (1) cap saturation: the commanded target exceeds the engaged set's rated capacity.
	saturated := target > sumCaps+fastLoopTierMargin

	// (2) under-delivery: the engaged set ACKs the command (no Modbus error) but cannot
	// physically deliver it - a faulted, low-SoC-derated or phase-limited unit. The Modbus
	// ACK can't reveal this; only the measured power can. The energy-balance target tracks
	// the persisting grid error (target ≈ measured + grid), so target − measured is the
	// undelivered watts. A dwell longer than the inverter ramp keeps normal ramp-up from
	// tripping it (and shortfallTicks is reset on engage so a freshly-engaged unit can ramp).
	// measured battery power is signed (negative = charging); convert to the delivered
	// magnitude in the active direction so the shortfall is correct for charge too -
	// otherwise target − measured is always large positive while charging and the trigger
	// fires every tick even when the battery is over-delivering.
	delivered := measured
	if plan.direction == batteryPlanCharge {
		delivered = -measured
	}
	if target-delivered > fastLoopShortfall {
		plan.shortfallTicks++
	} else {
		plan.shortfallTicks = 0
	}
	starved := plan.shortfallTicks >= fastLoopShortfallDwell

	if !saturated && !starved {
		return false
	}

	next := plan.standby[0]
	plan.standby = plan.standby[1:]
	plan.entries = append(plan.entries, next)
	delete(site.batteryStopped, next.name)
	plan.shortfallTicks = 0

	switch plan.direction {
	case batteryPlanCharge:
		site.batteryChargeActive = append(site.batteryChargeActive, next.name)
		site.batteryChargeTier = len(plan.entries)
	case batteryPlanDischarge:
		site.batteryDischargeActive = append(site.batteryDischargeActive, next.name)
		site.batteryDischargeTier = len(plan.entries)
	}

	reason := "saturated"
	if starved {
		reason = "under-delivering"
	}
	site.log.DEBUG.Printf("solar power (fast): tier up (%s), engaging %s (target %.0fW, delivered %.0fW, capacity %.0fW)", reason, next.name, target, delivered, sumCaps)
	return true
}

// batteryFastFlipCheck requests an out-of-band main-loop re-decision when the active
// direction has clamped to zero and the opposite direction is wanted past the dead band
// for a sustained dwell. It never flips direction itself - the main loop remains the sole
// authority, this only shortens when that decision runs. Caller holds batteryPlanMu.
func (site *Site) batteryFastFlipCheck(plan *batteryControlPlan, gridPower, battPower, target float64) {
	// only consider a flip once the active direction has clamped to zero (no work left)
	if target > fastLoopMinDelta {
		plan.flipSince = time.Time{}
		return
	}

	// opposite-direction need, using the offsets the main loop shipped for that side
	// (battery power convention: positive = discharging, negative = charging). This adds
	// back the battery's current power, exactly like the main loop's energy balance, so it
	// is ramp-invariant: the crossing is detectable immediately, without waiting for the
	// inverter to ramp the old direction down to zero (which would cost 3-4s for nothing).
	var oppositeNeed float64
	switch plan.direction {
	case batteryPlanDischarge: // opposite = charge
		oppositeNeed = -battPower - (gridPower + plan.oppositeGridOffset)
	case batteryPlanCharge: // opposite = discharge
		oppositeNeed = battPower + gridPower + plan.oppositeGridOffset - plan.oppositeEvExcluded
	}

	if oppositeNeed <= plan.threshold {
		plan.flipSince = time.Time{}
		return
	}

	// start the dwell timer on first detection; require it to stay wanted for the dwell
	if plan.flipSince.IsZero() {
		plan.flipSince = time.Now()
		return
	}
	if time.Since(plan.flipSince) < fastLoopFlipDwell {
		return
	}

	// cooldown: bound charge<->discharge thrash near the crossing (the scheduled main
	// tick still owns genuine flips during the window)
	if time.Since(site.lastBatteryFlipRequest) < fastLoopFlipCooldown {
		return
	}
	plan.flipSince = time.Time{}
	site.lastBatteryFlipRequest = time.Now()

	// non-blocking poke; the cap-1 channel coalesces bursts and a pending request
	select {
	case site.batteryReplanChan <- struct{}{}:
		site.log.DEBUG.Printf("solar power (fast): crossing detected (opposite need %.0fW > %.0fW dead band), requesting re-decision", oppositeNeed, plan.threshold)
	default:
	}
}

// batteryFastSend distributes target equally across the plan's entries, clamps to the
// per-battery cap and writes the power commands in parallel (each battery has its own
// Modbus connection). Returns the total actually commanded. Caller holds batteryPlanMu.
func (site *Site) batteryFastSend(plan *batteryControlPlan, target float64) float64 {
	share := target / float64(len(plan.entries))

	powers := make([]float64, len(plan.entries))
	var wg sync.WaitGroup
	for i, e := range plan.entries {
		p := share
		if e.cap > 0 && p > e.cap {
			p = e.cap
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			var err error
			if plan.direction == batteryPlanCharge {
				err = e.ctrl.SetBatteryChargePower(p)
			} else {
				err = e.ctrl.SetBatteryDischargePower(p)
			}
			if err != nil {
				site.log.ERROR.Printf("solar power (fast): %s: %v", e.name, err)
				return
			}
			powers[i] = p
		}()
	}
	wg.Wait()

	var commanded float64
	for _, p := range powers {
		commanded += p
	}

	plan.lastWrite = time.Now()
	return commanded
}
