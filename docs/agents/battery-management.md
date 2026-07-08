# Battery Management — Technical Reference

This document describes the full battery management stack implemented in this fork of evcc, covering both the upstream features and the custom extensions added for multi-battery RS485 control (Marstek Venus E Gen 3).

---

## 1. Battery Modes

The site maintains a `batteryMode` that is sent to each battery via `SetBatteryMode()` (the `BatteryController` API). Three modes are defined:

| Mode | Value | Effect |
|------|-------|--------|
| Normal | 1 | Battery uses its own internal algorithm (anti-feed / self-consumption) |
| Hold | 2 | RS485 control active; charge/discharge direction held at Stop (0W) |
| Charge | 3 | Force-charge at rated power until maxSoc is reached |

**Hold** is used as the "RS485 enabled, waiting for per-tick power commands" state. Without Hold, the inverter resets to Normal between ticks and ignores power commands.

---

## 2. Battery Mode Selection (`requiredBatteryMode`)

Each tick the system evaluates which mode to apply, in priority order:

1. **External control** (`batteryModeExternal`) — set via MQTT/HTTP, overrides everything until cleared
2. **External reset** — when external mode is cleared, forces Normal once to hand control back
3. **Grid charge active** — forces Charge mode (charges from grid at cheap tariff)
4. **Discharge control active** — forces Hold (prevents discharge during fast/planned EV charging)
5. **Solar control active** — keeps Hold so RS485 commands own every tick
6. **Modified mode cleanup** — returns to Normal if mode was previously modified

---

## 3. Solar Control (`batterySolarControl`)

When enabled, the site drives `SetBatteryChargePower` / `SetBatteryDischargePower` on each battery every tick via the `BatteryPowerController` API. This gives watt-level control instead of binary charge/discharge.

### 3.1 Surplus Calculation

Two formulas are used depending on battery SoC relative to `prioritySoc`:

**Normal mode** (`soc >= prioritySoc`):
```
surplus = -sitePower
```

**Priority mode** (`soc < prioritySoc`):
```
surplus = -(batteryPower + gridPower)
         = pvPower - housePower - EV
```

The priority-mode formula is derived from the energy balance identity and is sign-convention agnostic — it works correctly regardless of whether the battery meter reports positive or negative values for charging (handles both standard evcc convention and inverted conventions like Marstek register 30006).

### 3.2 Control Decision (`threshold`)

The outer switch uses a combined threshold:
```
threshold = standbyPower(10W) + batteryControlDeadBand
```

| Condition | Action |
|-----------|--------|
| `surplus > threshold` | Charge batteries from solar surplus |
| `sitePower > threshold` | Discharge batteries to cover home deficit |
| otherwise | Stop all (balanced / idle) |

---

## 4. Dead Band (`batteryControlDeadBand`)

An additional threshold on top of `standbyPower` (10W) before the system starts or continues charge/discharge. Prevents the control loop from reacting to small measurement noise around the zero-grid setpoint.

- **Default**: 0W (backward-compatible; effective threshold = 10W)
- **Recommended**: 50–100W for installations with noisy grid meters
- **API**: `POST /batterycontroldeadband/{value}` | MQTT `batteryControlDeadBand`

---

## 5. Tiered Activation (`computeTier`)

Engages enough batteries to handle the target power while avoiding sub-threshold per-unit commands that inverters silently ignore (e.g. Marstek <50W). The tier *count* is sized off a **fraction of each battery's rated power** — `batteryTierFraction = 0.5` (`site_battery.go`) — not the full rating:

```
tierCount = ceil(target / (batteryTierFraction × ratedPerBat))   # clamped [1, nBatteries]
```

Sizing on 50% of the rating spreads load onto more units earlier. This is deliberate: with single-phase inverters it puts power on **more phases** (helps phase balance), and each unit runs at a more efficient **partial load**; the trade-off is each extra inverter's standby draw. Very low power still lands on tier 1, so it never over-splits into ignored sub-50W commands. The **per-battery power cap stays the full rating** (`GetPowerLimits`), so a unit can still ramp to its limit when others drop out.

### Tier boundaries (example: 3 × 2500W rated, `batteryTierFraction = 0.5` → 1250W tier target)

| Tier | Batteries active | Target range |
|------|-----------------|--------------|
| 1 | 1 | 0 – 1250W |
| 2 | 2 | 1250 – 2500W |
| 3 | 3 | > 2500W |

(So a 2000W target engages 2 units at ~1000W each, rather than one unit at 2000W.)

### Hysteresis (15% dead band)

To prevent rapid tier switching when power hovers near a boundary (boundaries scale with the tier target, i.e. `batteryTierFraction × ratedPerBat`):
- **Switch up**: only when target > current-tier capacity × 1.15
- **Switch down**: only when target < previous-tier capacity × 0.85
- **Large jump (> 1 tier)**: responds immediately without dead band

Tier state (`batteryChargeTier`, `batteryDischargeTier`) is persisted between ticks and reset to 0 on startup.

---

## 6. Battery Selection within a Tier

Within the active tier, batteries are selected by SoC to naturally balance the pack over time:

- **Charging**: select the N **lowest-SoC** batteries (fill the most depleted first)
- **Discharging**: select the N **highest-SoC** batteries (drain the fullest first)

Non-selected batteries receive a stop command (both `SetBatteryChargePower(0)` and `SetBatteryDischargePower(0)`).

### Fallback (no `BatteryPowerLimiter`)

For batteries without power limits, the original flat minimum threshold is used: if the per-battery share would be below 50W (the minimum Marstek acts on), the full surplus is concentrated on the single best candidate.

---

## 7. Sticky Selection (SoC Hysteresis)

Re-selecting batteries purely by SoC on every tick causes ping-pong oscillation when batteries have similar SoC (e.g., 70% vs 69%). Each switch generates unnecessary Modbus writes and inverter ramp-up cycles.

### How it works

The active battery set is persisted between ticks (`batteryChargeActive`, `batteryDischargeActive`). On each tick:

1. If the stored set is still valid (same size, all names still in pool): keep it
2. Check if any non-active battery is **more than 3% better** than the worst battery in the active set:
   - Charging: candidate SoC < worst active SoC − 3%
   - Discharging: candidate SoC > worst active SoC + 3%
3. If yes: swap that one battery in (one swap per tick maximum)
4. If no: keep the current set unchanged

**Effect**: a battery holds its role until another unit is clearly better — no flipping due to integer SoC quantisation noise.

### Reset conditions
- Tier size changes (a battery joins or leaves the pool via minSoc/maxSoc)
- `batterySolarControl` toggles off/on

---

## 8. Charge Tapering

Linearly reduces charge power in the last 5% of SoC before `maxSoc`. Mimics the CC/CV charging profile that protects lithium cells from stress near full charge.

```
taperFactor = (maxSoc - currentSoc) / chargeTaperRange   (clamped to minimum 0.25)
chargePower = requestedPower × taperFactor
```

- **Taper range**: 5% SoC below maxSoc
- **Minimum factor**: 25% of requested power (never fully stopped by taper)
- **Per-battery**: applied individually using each battery's `BatterySocLimiter.GetSocLimits()`
- Applied after the hard-cap from `BatteryPowerLimiter`
- **Skipped during LFP calibration**: when `batteryCalibrationCharge` is active, tapering is bypassed entirely so batteries charge at full surplus power all the way to 100%

---

## 9. Priority SoC (`prioritySoc`)

When battery SoC is below this threshold:

- The surplus formula switches to the energy-balance variant `-(batPow + gridPow)` for sign-convention robustness
- Discharge is **not** blocked (this is a charging-priority concept only, not a discharge gate)
- The battery gets first claim on solar surplus before EV charging is allowed (handled upstream in `sitePower` calculation)

---

## 10. Buffer SoC (`bufferSoc` / `bufferStartSoc`)

Controls battery-supported EV charging:

- **`bufferSoc`**: when battery SoC is above this level, battery power is included in the available budget for EV charging even without solar surplus. Only *sustains* an already-running charge (`batteryBuffered && lp.charging()`); it never starts one.
- **`bufferStartSoc`**: EV charging from battery only starts when SoC exceeds this level (hysteresis to prevent immediately draining a partially-charged battery)

Both are only evaluated when `soc >= prioritySoc` — below that, [§9](#9-priority-soc-prioritysoc) takes precedence and the buffer logic never runs (`site.go`, `sitePower`).

### Why "battery covers the EV" is self-limiting in PV mode

Above `bufferSoc` the fast loop sets `dischargeEvExcluded = 0`, so the EV's load lands in the discharge target. This does **not** mean the battery backfills the charger's full power. `sitePower` includes `batteryPower` (discharging is positive), so battery discharge *raises* `sitePower`, which drives `targetCurrent` down in `pvMaxCurrent` until it hits the `minCurrent` floor that `batteryBuffered` itself pins. The loop settles with the charger at `minCurrent` and the battery covering that floor minus any solar surplus.

Consequences:

- A watt cap on the battery's EV contribution would be meaningless here: above `minCurrent` it never binds (surplus exists, so the discharge target is ~0), and below `minCurrent` it is unhonorable (the charger cannot go lower — the only way down is to pause it, which is what `bufferSoc = off` already does).
- The one mode where `evPower` is *not* surplus-bounded is Now/plan charging, which bypasses `pvMaxCurrent` entirely. That case is governed by `batteryDischargeControl` ([§11](#11-discharge-control-batterydischargecontrol)), not by `bufferSoc`.

`bufferSoc` off is encoded as `0` (Go: `bufferSoc > 0 && ...`) or `100` (UI default and `|| 100` fallback); both make the gate unsatisfiable.

---

## 11. Discharge Control (`batteryDischargeControl`)

When enabled, modifies battery discharge behaviour per-charger:

- **Fast/planned charging** (car connected StatusB+, mode=Now or planActive or minSocNotReached): the power consumed by **those specific chargers** (`evPowerFast`) is excluded from the discharge target — grid covers their load. Other chargers not in fast/planned mode and house loads are still covered by the battery. StatusC is not required so phase-negotiation transitions don't momentarily re-enable this protection.
- **Smart cost active**: car is actually charging (StatusC) and the current tariff rate is below the smart cost limit — full EV power excluded.

**Key behaviour**: discharge control is independent of `bufferSoc`. When the toggle is on and a fast/planned charger is active, battery does not cover that charger's load regardless of SoC level. It only covers house loads and other (non-fast) EV chargers, which is independent of this flag.

### How `dischargeEvExcluded` is derived

The main loop computes a single value in `buildBatterySnapshot` and publishes it on the snapshot; the fast loop subtracts it from every discharge target (`dischargeTarget = Σbatt + grid + dischargeOffset − dischargeEvExcluded`).

1. `evPowerFast` = Σ `GetChargePower()` over loadpoints where `GetStatus() != StatusA && IsFastChargingActive()`
2. `evPower` = Σ `GetChargePower()` over all non-heating loadpoints

```go
if site.dischargeControlActive(rate) {
    dischargeEvExcluded = evPowerFast          // (1) discharge control wins
} else if !(site.bufferSoc > 0 && site.battery.Soc > site.bufferSoc) {
    dischargeEvExcluded = evPower              // (2) below bufferSoc: battery refuses the EV
}                                              // (3) else 0: battery covers the EV
```

The three outcomes, in precedence order:

| Condition | `dischargeEvExcluded` | Battery covers |
| --- | --- | --- |
| `dischargeControlActive` | `evPowerFast` | house + non-fast chargers |
| else, SoC ≤ `bufferSoc` (or unset) | `evPower` | house only |
| else (SoC > `bufferSoc`) | `0` | house + EV (bounded by `minCurrent`, see [§10](#10-buffer-soc-buffersoc--bufferstartsoc)) |

`dischargeControlActive` is checked **first**, so when the toggle is on and a fast/planned charger is active the battery refuses that charger's load regardless of SoC — `bufferSoc` cannot override it.

There is no separate "battery may power the EV" toggle and none is needed: `bufferSoc` governs the PV case (and self-limits), `batteryDischargeControl` governs the fast-charge case. A third control would be a third source of truth over the same decision.

If neither `dischargeTarget` nor `chargeTarget` exceeds `snap.threshold` (`standbyPower + batteryControlDeadBand`), the fast loop commits to idle and stops all batteries.

---

## 12. Grid Charge (`batteryGridChargeLimit`)

When the current grid tariff price is at or below this limit, forces Charge mode (charges battery from grid at rated power). Useful for time-of-use tariffs.

---

## 13. MaxSoc Enforcement

When Charge mode is active, `applyBatteryMode` checks each battery's SoC against its `maxSoc` limit (via `BatterySocLimiter`). If any battery has reached `maxSoc`, the mode is switched to Hold to stop charging that unit.

In solar control mode, the tiered selection also filters out batteries that have reached `maxSoc` (moved to the `full` list and stopped).

---

## 14. MinSoc Enforcement

`minSoc` is a **hard discharge floor — enforced no matter what**. No battery discharges below its configured minimum under any meter, read, or control-state failure.

In the discharge case, batteries whose SoC is at or below their `minSoc` (from `BatterySocLimiter`) are moved to the `empty` list and stopped, excluded from the active discharge tier.

**Fail closed.** The floor check treats an *unknown* SoC the same as being at the floor:

- If a battery's SoC read fails for a cycle, it is moved to `empty` and not discharged — a transient read glitch can never drain the pack below min. (Earlier behaviour failed *open*: an unreadable SoC let discharge continue, which could drain a pack to 0%.)
- When the normal solar-control tick is skipped (e.g. site power unavailable), the loop no longer simply holds the last setpoints — which would keep a discharging battery running with no floor re-check. Instead `enforceBatteryMinSoc()` runs every such tick: for each battery it forces `SetBatteryDischargePower(0)` whenever SoC is at/below `minSoc` **or** cannot be read. Charging is left untouched so solar can still recover the pack.

---

## 15. Command Ordering & Latency

The control loop minimises the time between the grid measurement and the command reaching the active battery, and avoids gaps during battery handoffs.

### Safe swap ordering

When sticky selection swaps one battery for another, the command is sent to the **incoming** battery first and its Modbus result is checked:

1. Incoming battery receives its charge/discharge command
2. **Failure** (Modbus error): outgoing battery keeps running and receives the share as a one-tick fallback; the next tick re-evaluates selection normally
3. **Success** (Modbus ACK): the outgoing battery's fate depends on direction:
   - **Discharge swap**: the outgoing battery is *not* stopped this tick — a Modbus ACK only confirms the register write, while the incoming inverter still needs seconds to ramp up. The outgoing battery keeps covering the load for one more tick (stopped on the next tick via the regular non-selected path). The overlap briefly exports to grid (safe); stopping immediately would import during the ramp.
   - **Charge swap**: the outgoing battery is stopped immediately (deferred). Here the asymmetry reverses: a ramp gap merely exports surplus (safe), while an overlap would charge both batteries and import from grid (unsafe).

### Per-cycle SoC cache

All battery SoC values are read **once per control cycle** into a cache. Selection loops and sort comparators perform map lookups instead of issuing repeated Modbus reads (previously the sort comparator re-read the same battery's SoC on every comparison).

### Deferred stops

Stop commands for non-selected batteries (full, empty, outside the tier) are queued and executed **after** the active batteries have received their power commands. The Modbus writes for inactive units stay off the critical path, so the active battery reacts to a load change one or more seconds sooner. During a tier shrink this causes a brief overlap (remaining battery ramps up before the other stops) which errs toward grid export — the safe direction.

### Failed meter read guard

When the grid meter read fails (`sitePower` cannot be computed), the solar power control **skips the tick entirely** instead of acting on a zero value — a zero would be mistaken for "balanced" and stop all batteries for one tick, dropping the load onto the grid. Batteries hold their last setpoints until the next successful read. Battery *mode* handling still runs on such ticks.

### Redundant stop suppression

A battery that is already stopped is not re-stopped every tick. `stopBatteries` tracks ticks-since-last-stop per battery (`batteryStopped`) and skips the Modbus writes while the battery remains stopped, re-sending the stop every `stopRefreshTicks` (10) ticks as a watchdog heartbeat so RS485 control stays alive. Any active power command clears the tracking so the next stop is always sent immediately; a failed stop is retried on the next tick.

> **Note:** §15 describes mechanisms that now live inside the fast loop (§16) — the SoC read is the snapshot's, the stops are `stopBatteries`, and the failed-meter-read / single-writer / command-ordering concerns no longer apply in their old form (the fast loop is the sole writer and reads its own grid). Kept for the underlying rationale.

---

## 16. Battery Control Architecture (snapshot + fast controller)

The battery is controlled by a **1 s fast loop** (`core/site_battery_fast.go`) that owns *every* power decision. The main loop is a slow supervisor: it sets the battery **mode** and publishes a **snapshot**; it commands no power and picks no direction. This is the OpenEMS-style continuous-setpoint model — direction is simply the sign of the energy-balance need, so there is no separate charge/discharge decision to wait on and no flip-latency to patch.

| | Main loop (supervisor) | Fast loop (controller, 1 s) |
|---|---|---|
| Battery mode (Hold/Charge/Normal), EV, tariffs, calibration | ✔ | — |
| Snapshot: per-battery SoC/limits/caps + config | ✔ builds each cycle | consumes |
| Direction, tiering, sticky selection, swaps | — | ✔ |
| Power commands + stops | — | ✔ every tick |

**Snapshot contract** (`batterySnapshot`, built by `buildBatterySnapshot`, swapped under `batteryPlanMu`): per-battery `ctrl/meter/name`, cached `soc`, `min/maxSoc`, `chargeCap/dischargeCap`, plus `chargeOffset` / `dischargeOffset` / `dischargeEvExcluded` / `threshold` and the pool/tiering/sticky/tapering/calibration flags. No power, no direction. SoC is read once per main cycle — it moves ~0.02 %/5 s, so the fast loop selects/tiers off the snapshot without live SoC reads.

**Parking.** The snapshot is cleared (and the fast loop parks) when solar control is off, or when a higher-precedence controller overrides it — grid charge, or external/API battery mode. Those overrides mirror the precedence in `requiredBatteryMode`, so the fast loop never fights a controller that owns the battery.

When solar control is switched **fully off**, the main loop first stops every battery once before nil'ing the snapshot. This matters: the last actively-driven battery still holds the setpoint the fast loop wrote, and `SetBatteryMode(Normal)` only flips the mode register, not the power register — without the explicit stop it would keep charging/discharging indefinitely. The stop runs under `batteryPlanMu`, so the fast tick cannot run concurrently and the fast loop remains the sole power writer. On an *override* transition no stop is sent: the incoming controller re-commands power every cycle, so a stop would only fight it.

**Fast tick** (`batteryFastTick`): read fresh grid → stale-grid guard → parallel battery power reads → sampling-skew guard → compute both energy-balance targets:
- `dischargeTarget = Σbatt + grid + dischargeOffset − dischargeEvExcluded`
- `chargeTarget    = −Σbatt − (grid + chargeOffset)`

Both add back the measured battery power, so they are **ramp-invariant** (this was the fix for the early full-scale oscillation). Direction = whichever exceeds `threshold` (idle if neither). Then `fastControl` runs the full per-direction pipeline off the snapshot: eligibility filter (charge drops maxSoc units unless calibrating; discharge drops minSoc units and **fails closed** on an unreadable SoC), `computeTier` (hysteresis, sized at `batteryTierFraction` of rated), sticky selection with 3 % swap threshold, per-battery cap + charge taper, **parallel writes**, then deferred stops for the rest.

**Direction arbitration** (`batteryArbitrateDirection`): same-direction and to/from-idle changes pass through immediately; a charge↔discharge **reversal** must persist for `fastLoopFlipDwell` (1 s) and is spaced by an adaptive backoff (`batteryFlipBackoff`, 2 s doubling to 60 s on repeated reversals, reset after a `fastLoopFlipCalm` 120 s calm gap). While a reversal is pending the committed direction is held and its target clamps to zero, so the battery just ramps down.

**Meter guards:** skip on a stale grid register (identical reading) and on sampling skew (`|Δbatt| > 100 W` while `|Δgrid + Δbatt| > 100 W`). Guard state lives on the `Site` across ticks.

**Why this deleted a lot of code:** running the full selection every 1 s off a measured, ramp-invariant target means `computeTier` naturally handles both tier-up (saturation) and under-delivery, and direction falls out of the sign. So the previous bridge machinery — crossing detector, out-of-band `replanBattery` poke, dual offsets, explicit tier-up/standby, shortfall dwell, and the single-writer carve-outs — is **gone**; one controller, one cadence, one place for the runtime state.

**Known simplifications vs the previous split:** the swap safe-handoff keeps the *ordering* (incoming commanded before outgoing stopped, since writes precede deferred stops) but drops the Modbus-ACK check and the one-tick discharge overlap — a swap may cause a brief blip, acceptable as swaps are infrequent and SoC-driven. Idle↔active is instant (no dwell); if power hovers at the ~10 W threshold this could toggle at 1 s — raise `batteryControlDeadBand` if observed.

---

## Configuration Summary

| Setting | API | MQTT | Default | Description |
|---------|-----|------|---------|-------------|
| `batterySolarControl` | POST `/batterysolarcontrol/{bool}` | `batterySolarControl` | false | Enable watt-level solar charge/discharge control |
| `batteryControlDeadBand` | POST `/batterycontroldeadband/{W}` | `batteryControlDeadBand` | 0 | Extra dead band before starting charge/discharge |
| `batteryDischargeControl` | POST `/batterydischargecontrol/{bool}` | `batteryDischargeControl` | false | Prevent discharge during fast/smart charging |
| `batteryGridChargeLimit` | POST `/batterygridchargelimit/{price}` | `batteryGridChargeLimit` | null | Charge from grid when tariff ≤ this price |
| `batteryMode` | POST `/batterymode/{mode}` | `batteryMode` | normal | External mode override (normal/hold/charge) |
| `prioritySoc` | POST `/prioritysoc/{%}` | `prioritySoc` | 0 | Battery charging priority threshold |
| `bufferSoc` | POST `/buffersoc/{%}` | `bufferSoc` | 0 | SoC above which battery supports EV charging |
| `bufferStartSoc` | POST `/bufferstartsoc/{%}` | `bufferStartSoc` | 0 | SoC threshold to start battery-supported charging |

### Internal constants (not configurable at runtime)

| Constant | Value | Description |
|----------|-------|-------------|
| `standbyPower` | 10W | Minimum power considered non-zero |
| `chargeTaperRange` | 5% SoC | SoC band in which charge is tapered |
| `chargeMinFactor` | 25% | Minimum taper factor at maxSoc |
| Tier hysteresis | 15% | Dead band around tier-switch boundaries |
| Sticky SoC threshold | 3% | Minimum SoC difference to swap active battery |
| `stopRefreshTicks` | 10 ticks | Heartbeat interval for re-sending stop to stopped batteries |
