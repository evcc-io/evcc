package core

import (
	"errors"
	"math"
	"sort"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/evcc-io/evcc/util/config"
)

// chargeTaperRange is the SoC band (percentage points) before maxSoc in which the
// charge power is linearly tapered down to chargeMinFactor. This mimics the CC/CV
// profile that protects cells from stress near full charge. Some inverters enforce
// this internally; the software taper is a safety net when running in RS485 mode.
const (
	chargeTaperRange = 5.0  // begin tapering this many % below maxSoc
	chargeMinFactor  = 0.25 // taper down to 25% of requested power at maxSoc
	stopRefreshTicks = 10   // re-send stop to an already-stopped battery every N ticks (watchdog heartbeat)
	// batteryTierFraction sizes the tier count off a fraction of each battery's rated
	// charge/discharge power rather than the full rating, so load spreads onto more units
	// earlier (e.g. 2000W onto 2 of 3 at 50%). Spreading covers more phases (single-phase
	// inverters) and runs each unit at a more efficient partial load; the cost is each extra
	// inverter's standby draw. The per-battery power cap stays the full rating, so a unit can
	// still ramp to its limit when others drop out.
	batteryTierFraction = 0.5
)

// computeTier returns the number of batteries to activate given the current power target,
// per-battery rated power, the previously-used tier (for hysteresis), and the number
// of available batteries.
//
// Why tiering?
// Splitting a small target equally across all batteries often results in per-unit
// commands below the inverter's minimum effective power (e.g. Marstek ignores <50 W).
// Tiering uses the minimum number of batteries so each unit stays at or below its
// rated capacity — which also keeps each inverter at a more efficient operating point.
//
// Hysteresis (15% dead band):
// Without hysteresis the tier would flip on every tick when the target hovers near a
// boundary. The dead band means we switch up only when the target clearly exceeds the
// current tier's capacity (×1.15), and switch down only when it clearly falls below
// the previous tier's capacity (×0.85). Jumps of more than one tier skip the dead band
// and respond immediately.
func computeTier(target, maxPerBat float64, currentTier, maxTier int) int {
	if maxPerBat <= 0 || maxTier <= 1 {
		return maxTier
	}

	const hysteresis = 0.15

	// minimum batteries needed so each handles at most maxPerBat
	raw := int(math.Ceil(target / maxPerBat))
	if raw < 1 {
		raw = 1
	}
	if raw > maxTier {
		raw = maxTier
	}

	// first call: jump directly to the correct tier with no hysteresis
	if currentTier == 0 {
		return raw
	}

	diff := raw - currentTier
	if diff > 1 || diff < -1 {
		// large change (>1 tier): respond immediately without waiting for dead band
		return raw
	}

	// single-tier transitions: enforce hysteresis dead band
	if diff == 1 && target > float64(currentTier)*maxPerBat*(1+hysteresis) {
		return currentTier + 1 // clearly above current tier's capacity
	}
	if diff == -1 && target < float64(currentTier-1)*maxPerBat*(1-hysteresis) {
		return currentTier - 1 // clearly below previous tier's capacity
	}
	return currentTier
}

func batteryModeModified(mode api.BatteryMode) bool {
	return mode != api.BatteryUnknown && mode != api.BatteryNormal
}

func (site *Site) batteryConfigured() bool {
	return len(site.batteryMeters) > 0
}

func (site *Site) hasBatteryControl() bool {
	for _, dev := range site.batteryMeters {
		meter := dev.Instance()

		if api.HasCap[api.BatteryController](meter) {
			return true
		}
	}

	return false
}

// setBatteryMode sets the battery mode
func (site *Site) setBatteryMode(batMode api.BatteryMode) {
	site.batteryMode = batMode
	site.publish(keys.BatteryMode, batMode)
}

// SetBatteryMode sets the battery mode
func (site *Site) SetBatteryMode(batMode api.BatteryMode) {
	site.Lock()
	defer site.Unlock()

	site.log.DEBUG.Println("set battery mode:", batMode)

	if site.batteryMode != batMode {
		site.setBatteryMode(batMode)
	}

	if site.batteryModeExternal == api.BatteryUnknown {
		site.batteryModeExternalTimer = time.Time{}
	}
}

func (site *Site) updateBatteryMode(batteryGridChargeActive bool, rate api.Rate, sitePower float64, sitePowerValid bool) {
	batteryMode := site.requiredBatteryMode(batteryGridChargeActive, rate, sitePower)

	// put battery into hold mode when charging is active and HEMS dimmed
	fromToCharge := batteryMode == api.BatteryCharge || batteryMode == api.BatteryUnknown && site.batteryMode == api.BatteryCharge
	if dimmed := hems.Dimmed(site.hems); fromToCharge && dimmed != nil && *dimmed {
		site.log.DEBUG.Println("battery mode: HEMS dimmed")
		batteryMode = api.BatteryHold
	}

	// NOTE: applyBatteryMode is always called when charge mode is active to validate max soc
	if modeChanged := batteryMode != api.BatteryUnknown; modeChanged || site.batteryMode == api.BatteryCharge {
		if err := site.applyBatteryMode(batteryMode); err == nil {
			if modeChanged {
				site.SetBatteryMode(batteryMode)
			}
		} else {
			site.log.ERROR.Println("battery mode:", err)
		}
	}

	// when solar control is active, drive power-level setters on capable battery meters.
	// skip on failed meter read: a zero sitePower would be mistaken for "balanced" and
	// stop all batteries for a tick — holding the last setpoints is the safe behavior.
	if site.batterySolarControl {
		if sitePowerValid {
			site.applyBatterySolarPower(rate, sitePower)
		} else {
			// Holding the last setpoints would keep a discharging battery running
			// with no min-soc re-check. Enforce the hard floor instead so a stuck
			// meter read can never drain the pack below min.
			site.log.DEBUG.Println("solar power: skipping tick, site power unavailable")
			site.enforceBatteryMinSoc()
		}
	}
}

// enforceBatteryMinSoc is the hard discharge floor. It runs on every tick where
// the normal solar control loop does not (e.g. site power unavailable),
// guaranteeing that no battery discharges below its configured minimum soc
// regardless of meter or read failures. Charging is left untouched so solar can
// still recover the pack. SoC that cannot be read fails closed (discharge
// stopped).
func (site *Site) enforceBatteryMinSoc() {
	for _, dev := range site.batteryMeters {
		ctrl, ok := api.Cap[api.BatteryPowerController](dev.Instance())
		if !ok {
			continue
		}

		atFloor := true // fail closed: unknown soc → stop discharge
		if bat, hasBat := api.Cap[api.Battery](dev.Instance()); hasBat {
			if soc, err := bat.Soc(); err == nil {
				var minSoc float64
				if limiter, hasLimiter := api.Cap[api.BatterySocLimiter](dev.Instance()); hasLimiter {
					minSoc, _ = limiter.GetSocLimits()
				}
				atFloor = soc <= minSoc
			}
		}

		if atFloor {
			if err := ctrl.SetBatteryDischargePower(0); err != nil {
				site.log.ERROR.Printf("battery min-soc floor: %v", err)
			}
		}
	}
}

// applyBatterySolarPower calls SetBatteryChargePower / SetBatteryDischargePower on each battery
// meter that implements BatteryPowerController, proportional to the solar surplus or deficit.
func (site *Site) applyBatterySolarPower(rate api.Rate, sitePower float64) {
	// serialize against the fast loop and publish the control plan for it on exit;
	// paths that don't fill the plan leave it idle, which parks the fast loop
	site.batteryPlanMu.Lock()
	defer site.batteryPlanMu.Unlock()

	// seed the fast loop's meter consistency guard from this tick's readings so the
	// first fast tick after the plan swap is guarded as well
	plan := &batteryControlPlan{
		created:   time.Now(),
		lastWrite: time.Now(),
		lastGrid:  site.gridPower,
		lastBatt:  site.battery.Power,
		lastValid: true,
	}
	var planWrote bool
	defer func() {
		// carry the heartbeat reference over plan swaps when this tick didn't write
		if old := site.batteryPlan; old != nil && !planWrote && plan.direction == old.direction {
			plan.lastWrite = old.lastWrite
		}
		site.batteryPlan = plan
		site.batteryLastDirection = plan.direction
	}()

	// Single-writer: while the fast loop is active it owns the power values of
	// already-active batteries. The main loop only issues power commands on
	// activation (battery was stopped), direction change, or swap handling -
	// re-commanding steady-state batteries from the main loop's meter snapshot
	// injects sampling-skew phantoms that the fast loop then has to correct.
	fastLoopActive := site.gridMeter != nil

	// measured per-battery power for seeding plan.total of write-skipped batteries
	devPower := make(map[string]float64, len(site.batteryMeters))
	if len(site.battery.Devices) == len(site.batteryMeters) {
		for i, dev := range site.batteryMeters {
			devPower[dev.Config().Name] = site.battery.Devices[i].Power
		}
	}

	var evPower, evPowerFast float64
	for _, lp := range site.loadpoints {
		if !lp.IsHeating() {
			p := lp.GetChargePower()
			evPower += p
			if lp.GetStatus() != api.StatusA && lp.IsFastChargingActive() {
				evPowerFast += p
			}
		}
	}
	// When battery has priority (soc below threshold), use the raw grid reading as the
	// surplus signal so the battery charges from actual solar export regardless of the
	// sitePower adjustment that throttles loadpoints. In normal mode sitePower is used
	// directly (negative = exporting = surplus available for battery).
	// Auto-disable calibration charge once aggregate SoC reaches 100 %.
	// If the battery BMS never reports exactly 100 %, the toggle can be turned off manually.
	if site.batteryCalibrationCharge && site.battery.Soc >= 100 {
		site.Lock()
		site.batteryCalibrationCharge = false
		site.Unlock()
		site.publish(keys.BatteryCalibrationCharge, false)
		site.log.DEBUG.Printf("battery calibration charge complete (soc %.0f%%)", site.battery.Soc)
	}

	surplus := -sitePower // positive = exporting (solar surplus)
	if site.battery.Soc < site.prioritySoc {
		// Derive true solar surplus via energy balance, independent of battery sign
		// convention (standard: positive=charging, or inverted: negative=charging):
		//   pvPower - housePower - EV = -(batteryPower + gridPower)
		// Converges to the correct setpoint within 1-2 ticks for all battery states.
		surplus = -(site.battery.Power + site.gridPower)
	}

	type entry struct {
		ctrl api.BatteryPowerController
		dev  config.Device[api.Meter]
	}

	// collect all capable controllers
	var all []entry
	for _, dev := range site.batteryMeters {
		if ctrl, ok := api.Cap[api.BatteryPowerController](dev.Instance()); ok {
			all = append(all, entry{ctrl, dev})
		}
	}
	if len(all) == 0 {
		return
	}

	if site.batteryStopped == nil {
		site.batteryStopped = make(map[string]int)
	}

	// stop the given batteries; skip units already stopped, re-sending the stop only
	// every stopRefreshTicks as a watchdog heartbeat to keep RS485 control alive
	stopAll := func(entries []entry) {
		for _, e := range entries {
			name := e.dev.Config().Name
			if n, ok := site.batteryStopped[name]; ok && n < stopRefreshTicks {
				site.batteryStopped[name] = n + 1
				continue
			}
			failed := false
			if err := e.ctrl.SetBatteryChargePower(0); err != nil {
				site.log.ERROR.Printf("battery charge power: %v", err)
				failed = true
			}
			if err := e.ctrl.SetBatteryDischargePower(0); err != nil {
				site.log.ERROR.Printf("battery discharge power: %v", err)
				failed = true
			}
			if failed {
				// retry next tick
				delete(site.batteryStopped, name)
			} else {
				site.batteryStopped[name] = 0
			}
		}
	}

	// read each battery's SoC once per cycle; selection loops and sort comparators
	// hit this cache instead of issuing repeated Modbus reads
	type socReading struct {
		soc float64
		ok  bool
	}
	socCache := make(map[string]socReading, len(all))
	for _, e := range all {
		if bat, ok := api.Cap[api.Battery](e.dev.Instance()); ok {
			soc, err := bat.Soc()
			socCache[e.dev.Config().Name] = socReading{soc, err == nil}
		}
	}

	// per-device SoC from cache; returns 0 and ok=false when not available
	deviceSoc := func(dev config.Device[api.Meter]) (float64, bool) {
		r := socCache[dev.Config().Name]
		return r.soc, r.ok
	}

	// Dead band: require surplus/deficit to exceed standbyPower + configurable dead band
	// before starting or continuing charge/discharge. Prevents the control loop from
	// reacting to small measurement noise around the zero-grid setpoint.
	threshold := standbyPower + site.batteryControlDeadBand

	// Direction-agnostic grid setpoints, computed once so the active branch and the
	// fast loop's opposite-direction crossing detector stay consistent. Charge ignores
	// residualPower below prioritySoc (energy-balance surplus already does); discharge
	// excludes fast/planned EV power that the battery must not cover.
	residual := site.GetResidualPower()
	chargeOffset := 0.0
	if site.battery.Soc >= site.prioritySoc {
		chargeOffset = residual
	}
	dischargeOffset := residual
	// mirrors the discharge branch's EV-exclusion exactly (kept in sync by construction):
	// discharge control excludes only fast/planned EV; below bufferSoc excludes all EV
	var dischargeEvExcluded float64
	if site.dischargeControlActive(rate) {
		dischargeEvExcluded = evPowerFast
	} else if !(site.bufferSoc > 0 && site.battery.Soc > site.bufferSoc) {
		dischargeEvExcluded = evPower
	}
	plan.threshold = threshold

	switch {
	case surplus > threshold:
		// filter to batteries that have not yet reached their max SoC.
		// When LFP calibration is active, skip the maxSoc check so all batteries
		// can charge to 100 % regardless of their configured upper limit.
		var active, full []entry
		var chargeSwapIn, chargeSwapOut entry
		var hasChargeSwap bool
		// stops are deferred until after the active batteries received their commands,
		// keeping the Modbus writes for inactive batteries off the critical path
		var deferStop []entry
		// eligible batteries excluded from the current tier; published as fast-loop
		// standby so it can engage them on saturation (tier-up)
		var standby []entry
		for _, e := range all {
			soc, ok := deviceSoc(e.dev)
			if !site.batteryCalibrationCharge {
				if limiter, hasLimiter := api.Cap[api.BatterySocLimiter](e.dev.Instance()); ok && hasLimiter {
					if _, maxSoc := limiter.GetSocLimits(); maxSoc > 0 && soc >= maxSoc {
						full = append(full, e)
						continue
					}
				}
			}
			_ = ok
			active = append(active, e)
		}
		deferStop = append(deferStop, full...)
		if len(active) == 0 {
			stopAll(all)
			break
		}

		if !site.batterySolarPool {
			// Per-battery mode: optionally activate minimum number of batteries (tiering)
			// and optionally keep selection stable across ticks (sticky).
			var maxChargePerBat float64
			for _, e := range active {
				if limiter, ok := api.Cap[api.BatteryPowerLimiter](e.dev.Instance()); ok {
					if c, _ := limiter.GetPowerLimits(); c > 0 && (maxChargePerBat == 0 || c < maxChargePerBat) {
						maxChargePerBat = c
					}
				}
			}

			if maxChargePerBat > 0 && site.batterySolarTiering {
				tierPerBat := maxChargePerBat * batteryTierFraction
				site.batteryChargeTier = computeTier(surplus, tierPerBat, site.batteryChargeTier, len(active))
				needed := site.batteryChargeTier

				if needed < len(active) {
					if site.batterySolarSticky {
						// Sticky selection: keep current set, swap only on significant SoC divergence.
						const socSwitchThreshold = 3.0
						prevSet := make(map[string]bool, len(site.batteryChargeActive))
						for _, n := range site.batteryChargeActive {
							prevSet[n] = true
						}
						var sel, cand []entry
						for _, e := range active {
							if prevSet[e.dev.Config().Name] {
								sel = append(sel, e)
							} else {
								cand = append(cand, e)
							}
						}
						if len(sel) != needed {
							sort.Slice(active, func(i, j int) bool {
								si, _ := deviceSoc(active[i].dev)
								sj, _ := deviceSoc(active[j].dev)
								return si < sj
							})
							sel = active[:needed]
							cand = active[needed:]
						} else {
							worstIdx, worstSoc := 0, 0.0
							for i, s := range sel {
								if soc, ok := deviceSoc(s.dev); ok && (i == 0 || soc > worstSoc) {
									worstSoc, worstIdx = soc, i
								}
							}
							for ci, c := range cand {
								if socC, ok := deviceSoc(c.dev); ok && worstSoc-socC > socSwitchThreshold {
									site.log.DEBUG.Printf("solar power: charge swap %s (%.0f%%) → %s (%.0f%%)",
										sel[worstIdx].dev.Config().Name, worstSoc,
										c.dev.Config().Name, socC)
									chargeSwapIn = c
									chargeSwapOut = sel[worstIdx]
									hasChargeSwap = true
									sel[worstIdx], cand[ci] = c, sel[worstIdx]
									break
								}
							}
						}
						site.batteryChargeActive = make([]string, len(sel))
						for i, e := range sel {
							site.batteryChargeActive[i] = e.dev.Config().Name
						}
						if hasChargeSwap {
							for _, c := range cand {
								if c.dev.Config().Name != chargeSwapOut.dev.Config().Name {
									deferStop = append(deferStop, c)
									standby = append(standby, c)
								}
							}
						} else {
							deferStop = append(deferStop, cand...)
							standby = append(standby, cand...)
						}
						active = sel
					} else {
						// No sticky: sort by SoC each tick, pick the N most depleted.
						sort.Slice(active, func(i, j int) bool {
							si, _ := deviceSoc(active[i].dev)
							sj, _ := deviceSoc(active[j].dev)
							return si < sj
						})
						deferStop = append(deferStop, active[needed:]...)
						standby = append(standby, active[needed:]...)
						active = active[:needed]
					}
					site.log.DEBUG.Printf("solar power: charge tier %d/%d — %.0fW surplus, %.0fW/bat tier target (%.0fW rated)", needed, len(all)-len(full), surplus, tierPerBat, maxChargePerBat)
				} else {
					site.batteryChargeActive = nil
				}
			} else if maxChargePerBat == 0 {
				// No BatteryPowerLimiter: concentrate on most-depleted battery if share is too small.
				const minEffectiveShare = 50.0
				share := surplus / float64(len(active))
				if share < minEffectiveShare && len(active) > 1 {
					bestIdx, bestSoc := 0, 101.0
					for i, e := range active {
						if soc, ok := deviceSoc(e.dev); ok && soc < bestSoc {
							bestSoc, bestIdx = soc, i
						}
					}
					var others []entry
					for i, e := range active {
						if i != bestIdx {
							others = append(others, e)
						}
					}
					deferStop = append(deferStop, others...)
					active = active[bestIdx : bestIdx+1]
					site.log.DEBUG.Printf("solar power: charge share %.0fW below %.0fW min, concentrating on lowest-soc battery (%.0f%%)", surplus/float64(len(all)-len(full)), minEffectiveShare, bestSoc)
				}
			}
			// if tiering is off but BatteryPowerLimiter is available: use all active batteries equally
		}
		// pool mode: use all active batteries equally (no selection)

		share := surplus / float64(len(active))
		var chargeSwapInFailed bool
		for _, e := range active {
			chargePower := share

			var capW float64
			if limiter, ok := api.Cap[api.BatteryPowerLimiter](e.dev.Instance()); ok {
				if maxCharge, _ := limiter.GetPowerLimits(); maxCharge > 0 {
					capW = maxCharge
					if chargePower > maxCharge {
						chargePower = maxCharge
					}
				}
			}

			taper := 1.0
			if site.batterySolarTapering && !site.batteryCalibrationCharge {
				if limiter, ok := api.Cap[api.BatterySocLimiter](e.dev.Instance()); ok {
					if _, maxSoc := limiter.GetSocLimits(); maxSoc > 0 {
						if soc, ok := deviceSoc(e.dev); ok && soc > maxSoc-chargeTaperRange {
							factor := (maxSoc - soc) / chargeTaperRange
							if factor < chargeMinFactor {
								factor = chargeMinFactor
							}
							chargePower *= factor
							taper = factor
						}
					}
				}
			}

			name := e.dev.Config().Name
			_, wasStopped := site.batteryStopped[name]
			isSwapIn := hasChargeSwap && name == chargeSwapIn.dev.Config().Name
			delete(site.batteryStopped, name)

			if fastLoopActive && !wasStopped && !isSwapIn && site.batteryLastDirection == batteryPlanCharge {
				// already active in the same direction: the fast loop owns the power
				// value; seed the plan total from the measured power instead
				plan.entries = append(plan.entries, batteryPlanEntry{e.ctrl, e.dev.Instance(), name, capW * taper})
				plan.total += math.Max(0, -devPower[name])
				continue
			}

			if err := e.ctrl.SetBatteryChargePower(chargePower); err != nil {
				site.log.ERROR.Printf("battery charge power: %v", err)
				if isSwapIn {
					chargeSwapInFailed = true
				}
			} else {
				planWrote = true
				plan.entries = append(plan.entries, batteryPlanEntry{e.ctrl, e.dev.Instance(), name, capW * taper})
				plan.total += chargePower
			}
		}
		// On swap ticks the plan stays idle for one main tick: inverter ramps make the
		// fast loop's commanded-power proxy unreliable until the handoff settles.
		if !hasChargeSwap {
			plan.direction = batteryPlanCharge
			plan.gridOffset = chargeOffset
			plan.oppositeGridOffset = dischargeOffset
			plan.oppositeEvExcluded = dischargeEvExcluded
		}
		if hasChargeSwap {
			if chargeSwapInFailed {
				site.log.WARN.Printf("solar power: charge swap failed, keeping %s", chargeSwapOut.dev.Config().Name)
				delete(site.batteryStopped, chargeSwapOut.dev.Config().Name)
				if err := chargeSwapOut.ctrl.SetBatteryChargePower(share); err != nil {
					site.log.ERROR.Printf("battery charge power fallback: %v", err)
				} else {
					planWrote = true
				}
			} else {
				deferStop = append(deferStop, chargeSwapOut)
			}
		}
		// publish standby candidates (lowest SoC engaged first) for fast-loop tier-up;
		// skipped on swap ticks where the plan stays idle
		if !hasChargeSwap && len(standby) > 0 {
			sort.Slice(standby, func(i, j int) bool {
				si, _ := deviceSoc(standby[i].dev)
				sj, _ := deviceSoc(standby[j].dev)
				return si < sj
			})
			for _, e := range standby {
				var capW float64
				if limiter, ok := api.Cap[api.BatteryPowerLimiter](e.dev.Instance()); ok {
					if c, _ := limiter.GetPowerLimits(); c > 0 {
						capW = c
					}
				}
				plan.standby = append(plan.standby, batteryPlanEntry{e.ctrl, e.dev.Instance(), e.dev.Config().Name, capW})
			}
		}
		site.log.DEBUG.Printf("solar power: charge %.0fW across %d/%d batteries", share*float64(len(active)), len(active), len(all))
		stopAll(deferStop)

	case sitePower > threshold:
		// Compute discharge target by subtracting EV power the battery should NOT cover.
		// When discharge control is active (toggle on + fast/planned charger), only fast-charging
		// EV power is excluded — other chargers and house loads remain battery-covered.
		// When battery SoC is below bufferSoc, all EV power is excluded regardless.
		batteryBufferedEv := site.bufferSoc > 0 && site.battery.Soc > site.bufferSoc
		dischargeTarget := sitePower
		var evExcluded float64
		if site.dischargeControlActive(rate) {
			dischargeTarget -= evPowerFast
			evExcluded = evPowerFast
		} else if !batteryBufferedEv {
			dischargeTarget -= evPower
			evExcluded = evPower
		}
		if dischargeTarget <= standbyPower {
			stopAll(all)
			site.log.DEBUG.Printf("solar power: discharge prevented (EV deficit only), stop")
			break
		}
		var active, empty []entry
		var dischargeSwapIn, dischargeSwapOut entry
		var hasDischargeSwap bool
		// stops are deferred until after the active batteries received their commands,
		// keeping the Modbus writes for inactive batteries off the critical path
		var deferStop []entry
		// eligible batteries excluded from the current tier; published as fast-loop
		// standby so it can engage them on saturation (tier-up)
		var standby []entry
		for _, e := range all {
			soc, ok := deviceSoc(e.dev)
			// Hard min-soc floor: never discharge below the configured minimum.
			// Fail closed — if the SoC read failed this cycle, treat the battery as
			// empty so a transient read glitch can never drain the pack below min.
			if !ok {
				empty = append(empty, e)
				continue
			}
			if limiter, hasLimiter := api.Cap[api.BatterySocLimiter](e.dev.Instance()); hasLimiter {
				if minSoc, _ := limiter.GetSocLimits(); soc <= minSoc {
					empty = append(empty, e)
					continue
				}
			}
			active = append(active, e)
		}
		deferStop = append(deferStop, empty...)
		if len(active) == 0 {
			stopAll(all)
			break
		}

		if !site.batterySolarPool {
			var maxDischargePerBat float64
			for _, e := range active {
				if limiter, ok := api.Cap[api.BatteryPowerLimiter](e.dev.Instance()); ok {
					if _, d := limiter.GetPowerLimits(); d > 0 && (maxDischargePerBat == 0 || d < maxDischargePerBat) {
						maxDischargePerBat = d
					}
				}
			}

			if maxDischargePerBat > 0 && site.batterySolarTiering {
				tierPerBat := maxDischargePerBat * batteryTierFraction
				site.batteryDischargeTier = computeTier(dischargeTarget, tierPerBat, site.batteryDischargeTier, len(active))
				needed := site.batteryDischargeTier

				if needed < len(active) {
					if site.batterySolarSticky {
						const socSwitchThreshold = 3.0
						prevSet := make(map[string]bool, len(site.batteryDischargeActive))
						for _, n := range site.batteryDischargeActive {
							prevSet[n] = true
						}
						var sel, cand []entry
						for _, e := range active {
							if prevSet[e.dev.Config().Name] {
								sel = append(sel, e)
							} else {
								cand = append(cand, e)
							}
						}
						if len(sel) != needed {
							sort.Slice(active, func(i, j int) bool {
								si, _ := deviceSoc(active[i].dev)
								sj, _ := deviceSoc(active[j].dev)
								return si > sj
							})
							sel = active[:needed]
							cand = active[needed:]
						} else {
							worstIdx, worstSoc := 0, 101.0
							for i, s := range sel {
								if soc, ok := deviceSoc(s.dev); ok && (i == 0 || soc < worstSoc) {
									worstSoc, worstIdx = soc, i
								}
							}
							for ci, c := range cand {
								if socC, ok := deviceSoc(c.dev); ok && socC-worstSoc > socSwitchThreshold {
									site.log.DEBUG.Printf("solar power: discharge swap %s (%.0f%%) → %s (%.0f%%)",
										sel[worstIdx].dev.Config().Name, worstSoc,
										c.dev.Config().Name, socC)
									dischargeSwapIn = c
									dischargeSwapOut = sel[worstIdx]
									hasDischargeSwap = true
									sel[worstIdx], cand[ci] = c, sel[worstIdx]
									break
								}
							}
						}
						site.batteryDischargeActive = make([]string, len(sel))
						for i, e := range sel {
							site.batteryDischargeActive[i] = e.dev.Config().Name
						}
						if hasDischargeSwap {
							for _, c := range cand {
								if c.dev.Config().Name != dischargeSwapOut.dev.Config().Name {
									deferStop = append(deferStop, c)
									standby = append(standby, c)
								}
							}
						} else {
							deferStop = append(deferStop, cand...)
							standby = append(standby, cand...)
						}
						active = sel
					} else {
						sort.Slice(active, func(i, j int) bool {
							si, _ := deviceSoc(active[i].dev)
							sj, _ := deviceSoc(active[j].dev)
							return si > sj
						})
						deferStop = append(deferStop, active[needed:]...)
						standby = append(standby, active[needed:]...)
						active = active[:needed]
					}
					site.log.DEBUG.Printf("solar power: discharge tier %d/%d — %.0fW target, %.0fW/bat tier target (%.0fW rated)", needed, len(all)-len(empty), dischargeTarget, tierPerBat, maxDischargePerBat)
				} else {
					site.batteryDischargeActive = nil
				}
			} else if maxDischargePerBat == 0 {
				const minEffectiveShare = 50.0
				share := dischargeTarget / float64(len(active))
				if share < minEffectiveShare && len(active) > 1 {
					bestIdx, bestSoc := 0, 0.0
					for i, e := range active {
						if soc, ok := deviceSoc(e.dev); ok && soc > bestSoc {
							bestSoc, bestIdx = soc, i
						}
					}
					var others []entry
					for i, e := range active {
						if i != bestIdx {
							others = append(others, e)
						}
					}
					deferStop = append(deferStop, others...)
					active = active[bestIdx : bestIdx+1]
					site.log.DEBUG.Printf("solar power: discharge share %.0fW below %.0fW min, concentrating on highest-soc battery (%.0f%%)", dischargeTarget/float64(len(all)-len(empty)), minEffectiveShare, bestSoc)
				}
			}
		}

		share := dischargeTarget / float64(len(active))
		var dischargeSwapInFailed bool
		for _, e := range active {
			dischargePower := share

			var capW float64
			if limiter, ok := api.Cap[api.BatteryPowerLimiter](e.dev.Instance()); ok {
				if _, maxDischarge := limiter.GetPowerLimits(); maxDischarge > 0 {
					capW = maxDischarge
					if dischargePower > maxDischarge {
						dischargePower = maxDischarge
					}
				}
			}

			name := e.dev.Config().Name
			_, wasStopped := site.batteryStopped[name]
			isSwapIn := hasDischargeSwap && name == dischargeSwapIn.dev.Config().Name
			delete(site.batteryStopped, name)

			if fastLoopActive && !wasStopped && !isSwapIn && site.batteryLastDirection == batteryPlanDischarge {
				// already active in the same direction: the fast loop owns the power
				// value; seed the plan total from the measured power instead
				plan.entries = append(plan.entries, batteryPlanEntry{e.ctrl, e.dev.Instance(), name, capW})
				plan.total += math.Max(0, devPower[name])
				continue
			}

			if err := e.ctrl.SetBatteryDischargePower(dischargePower); err != nil {
				site.log.ERROR.Printf("battery discharge power: %v", err)
				if isSwapIn {
					dischargeSwapInFailed = true
				}
			} else {
				planWrote = true
				plan.entries = append(plan.entries, batteryPlanEntry{e.ctrl, e.dev.Instance(), name, capW})
				plan.total += dischargePower
			}
		}
		// On swap ticks the plan stays idle for one main tick: the outgoing battery still
		// covers part of the load during the overlap, which would mislead the fast loop's
		// commanded-power proxy and make it throttle the incoming battery.
		if !hasDischargeSwap {
			plan.direction = batteryPlanDischarge
			plan.evExcluded = evExcluded
			plan.gridOffset = dischargeOffset
			plan.oppositeGridOffset = chargeOffset
			plan.oppositeEvExcluded = 0
		}
		if hasDischargeSwap && dischargeSwapInFailed {
			site.log.WARN.Printf("solar power: discharge swap failed, keeping %s", dischargeSwapOut.dev.Config().Name)
			delete(site.batteryStopped, dischargeSwapOut.dev.Config().Name)
			if err := dischargeSwapOut.ctrl.SetBatteryDischargePower(share); err != nil {
				site.log.ERROR.Printf("battery discharge power fallback: %v", err)
			} else {
				planWrote = true
			}
		}
		// On successful swap the outgoing battery is intentionally NOT stopped this tick:
		// it keeps covering the load while the incoming inverter ramps up, and is stopped
		// on the next tick via the regular non-selected path. The brief overlap exports to
		// grid (safe); stopping immediately would import during the ramp. Charge swaps do
		// the opposite (stop immediately) since there the gap exports and overlap imports.
		// publish standby candidates (highest SoC engaged first) for fast-loop tier-up;
		// skipped on swap ticks where the plan stays idle
		if !hasDischargeSwap && len(standby) > 0 {
			sort.Slice(standby, func(i, j int) bool {
				si, _ := deviceSoc(standby[i].dev)
				sj, _ := deviceSoc(standby[j].dev)
				return si > sj
			})
			for _, e := range standby {
				var capW float64
				if limiter, ok := api.Cap[api.BatteryPowerLimiter](e.dev.Instance()); ok {
					if _, d := limiter.GetPowerLimits(); d > 0 {
						capW = d
					}
				}
				plan.standby = append(plan.standby, batteryPlanEntry{e.ctrl, e.dev.Instance(), e.dev.Config().Name, capW})
			}
		}
		site.log.DEBUG.Printf("solar power: discharge %.0fW across %d/%d batteries", share*float64(len(active)), len(active), len(all))
		stopAll(deferStop)

	default:
		stopAll(all)
		site.log.DEBUG.Printf("solar power: balanced, stop")
	}
}

// requiredBatteryMode determines required battery mode based on grid charge, rate, and site power
func (site *Site) requiredBatteryMode(batteryGridChargeActive bool, rate api.Rate, sitePower float64) api.BatteryMode {
	var res api.BatteryMode
	batMode := site.GetBatteryMode()
	extMode := site.GetBatteryModeExternal()

	var extModeReset bool
	if extMode == api.BatteryUnknown {
		site.Lock()
		extModeReset = !site.batteryModeExternalTimer.IsZero()
		site.Unlock()
	}

	keepUnlessModified := func(s api.BatteryMode) api.BatteryMode {
		return map[bool]api.BatteryMode{false: s, true: api.BatteryUnknown}[batMode == s]
	}

	switch {
	case !site.batteryConfigured():
		res = api.BatteryUnknown
	case extModeReset:
		// require normal mode to leave external control
		res = api.BatteryNormal
	case extMode != api.BatteryUnknown:
		// require external mode only once
		if extMode != batMode {
			res = extMode
		}
	case batteryGridChargeActive:
		res = keepUnlessModified(api.BatteryCharge)
	case site.dischargeControlActive(rate):
		res = keepUnlessModified(api.BatteryHold)
	case site.batterySolarControl:
		// Battery control: keep RS485 enabled (Hold) so applyBatterySolarPower owns every tick.
		// Normal mode would disable RS485 between ticks and hand control back to the inverter.
		res = keepUnlessModified(api.BatteryHold)
	case batteryModeModified(batMode):
		res = api.BatteryNormal
	}

	return res
}

// batteryMaxSocReached checks is battery has exceed max soc limit
func (site *Site) batteryMaxSocReached(dev config.Device[api.Meter]) (bool, error) {
	// Calibration charge bypasses the maxSoc limit so the battery charges to 100 %
	if site.batteryCalibrationCharge {
		return false, nil
	}

	meter := dev.Instance()

	batLimiter, ok := api.Cap[api.BatterySocLimiter](meter)
	if !ok {
		return false, nil
	}

	batSoc, ok := api.Cap[api.Battery](meter)
	if !ok {
		return false, errors.New("battery with soc limits must have soc")
	}

	soc, err := batSoc.Soc()
	if err != nil {
		return false, err
	}

	if _, max := batLimiter.GetSocLimits(); max > 0 && max < 100 && soc >= max {
		site.log.DEBUG.Printf("battery %s: limit soc reached (%.0f > %.0f)", deviceTitleOrName(dev), soc, max)
		return true, nil
	}

	return false, nil
}

// applyBatteryMode applies the mode to each battery
//
// api.BatteryCharge:
//
//	The current soc is validated against max soc.
//	In case max soc is reached, hold mode is applied.
func (site *Site) applyBatteryMode(mode api.BatteryMode) error {
	fromToCharge := mode == api.BatteryCharge || mode == api.BatteryUnknown && site.batteryMode == api.BatteryCharge

	for _, dev := range site.batteryMeters {
		meter := dev.Instance()

		batCtrl, ok := api.Cap[api.BatteryController](meter)
		if !ok {
			continue
		}

		// validate max soc
		if fromToCharge && mode != api.BatteryHold {
			ok, err := site.batteryMaxSocReached(dev)
			if err != nil && !errors.Is(err, api.ErrNotAvailable) {
				return err
			}

			// put battery into hold mode when soc limit reached
			if ok {
				// TODO do this only once
				mode = api.BatteryHold
			}
		}

		if mode != api.BatteryUnknown {
			if err := batCtrl.SetBatteryMode(mode); err == nil {
				site.log.DEBUG.Printf("set battery %s mode: %s", deviceTitleOrName(dev), mode)
			} else if !errors.Is(err, api.ErrNotAvailable) {
				return err
			}
		}
	}

	return nil
}

func (site *Site) tariffRates(usage api.TariffUsage) (api.Rates, error) {
	tariff := site.GetTariff(usage)
	if tariff == nil || tariff.Type() == api.TariffTypePriceStatic {
		return nil, nil
	}

	return tariff.Rates()
}

func (site *Site) smartCostActive(lp loadpoint.API, rate api.Rate) bool {
	limit := lp.GetSmartCostLimit()
	return limit != nil && !rate.IsZero() && rate.Value <= *limit
}

func (site *Site) batteryGridChargeActive(rate api.Rate) bool {
	limit := site.GetBatteryGridChargeLimit()
	return limit != nil && !rate.IsZero() && rate.Value <= *limit
}

func (site *Site) dischargeControlActive(rate api.Rate) bool {
	if !site.GetBatteryDischargeControl() {
		return false
	}

	for _, lp := range site.Loadpoints() {
		// fast/plan charging: car must be connected (StatusB+) but StatusC not required
		// so phase negotiation / ramp-up transitions don't momentarily re-enable discharge
		if lp.GetStatus() != api.StatusA && lp.IsFastChargingActive() {
			return true
		}
		// smart cost: only prevent discharge when current is actually flowing
		if lp.GetStatus() == api.StatusC && site.smartCostActive(lp, rate) {
			return true
		}
	}

	return false
}
