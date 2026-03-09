# Plan: Power-First Charger Control Architecture

## Problem

The loadpoint controls all chargers via **current (amps)**, even though the site provides surplus as **watts**. The flow is:

```
Site: sitePower (W) → Loadpoint: pvMaxCurrent → targetCurrent (A) → setLimit(A) → charger.MaxCurrent(A)
```

This creates three problems:

1. **Lossy round-trip for power-controlled devices.** Heat pumps, water heaters (MyPV, E.G.O., SG Ready, Heatpump) natively accept watts. Today the loadpoint converts W→A, sends amps, and the charger converts A→W back. The conversion uses global `Voltage=230V` and the loadpoint's phase count, which may not reflect reality for these devices.

2. **Leaky abstraction.** Power-controlled chargers must implement `loadpoint.Controller` and call `lp.GetPhases()` just to undo the W→A conversion. This couples charger implementations to loadpoint internals.

3. **Complexity.** The loadpoint mixes strategy logic (when/how much to charge) with charger control mechanics (current rounding, phase switching, circuit validation). Everything is expressed in current even though the natural unit is power.

## Current Architecture

### Power Flow

```
grid + battery + PV + excessDC - aux - flexible
              ↓
    site.sitePower() → sitePower (W)
              ↓
    lp.Update(sitePower, ...)
              ↓
    lp.pvMaxCurrent() → targetCurrent (A)  ← converts W to A, handles phase switching
              ↓
    lp.setLimit(current)                    ← rounds, validates, enables/disables
              ↓
    charger.MaxCurrent(A) or charger.MaxCurrentMillis(A)
```

### Key Functions (all in Loadpoint)

| Function | Responsibility |
|----------|---------------|
| `pvMaxCurrent()` | Converts sitePower→current, handles phase switching timers, PV enable/disable timers |
| `setLimit(current)` | Rounds current, applies circuit limits, calls charger, handles enable/disable + vehicle wake-up |
| `fastCharging()` | Forces 3p + max current |
| `effectiveCurrent()` | Returns actual charge current with hysteresis for soft vehicles (Zoe) |
| `effectiveMinCurrent()` / `effectiveMaxCurrent()` | Min/max from loadpoint config, vehicle, charger |
| `pvScalePhases()` | Phase switching decisions with timers |
| `roundedCurrent()` / `coarseCurrent()` | Integer amp rounding for basic chargers |

### The Coupling Problem

Power-controlled chargers today:
```go
// charger/mypv.go
func (wb *MyPv) MaxCurrentMillis(current float64) error {
    phases := 1
    if wb.lp != nil {
        if p := wb.lp.GetPhases(); p != 0 { phases = p }
    }
    power := uint16(voltage * current * float64(phases))  // undo the conversion
    return wb.setPower(power)
}
```

They receive current, query the loadpoint for phases, and convert back to power. This is the exact calculation the loadpoint already did in reverse.

## Proposed Architecture

### Core Idea

**Power is the universal language.** The loadpoint's strategy logic operates entirely in watts. A `ChargerController` interface translates power targets into charger-specific commands. Current control with phase switching is one implementation; direct power control is another.

### New Flow

```
site.sitePower() → sitePower (W)
        ↓
lp.Update(sitePower, ...)
        ↓
lp.pvMaxPower() → targetPower (W)  ← works entirely in watts, no phase awareness
        ↓
controller.SetPower(watts)
        ↓
┌─────────────────────────┬──────────────────────────────────┐
│ DirectPowerController   │ CurrentController                │
│                         │                                  │
│ circuit.ValidatePower() │ optimizePhases(power)            │
│ charger.MaxPower(W)     │ current = powerToCurrent(W, φ)  │
│ charger.Enable()        │ roundedCurrent(current)          │
│                         │ circuit.ValidateCurrent/Power()  │
│                         │ charger.MaxCurrentMillis(A)      │
│                         │ charger.Enable()                 │
│                         │ vehicle wake-up                  │
└─────────────────────────┴──────────────────────────────────┘
```

### Interface Definitions

```go
// core/chargercontroller/controller.go

// Controller translates power targets into charger commands.
// The loadpoint calls SetPower() with a target in watts;
// the controller handles the specifics of how to deliver that power.
type Controller interface {
    // SetPower sets the target charging power in watts.
    // 0 or below MinPower means disable.
    SetPower(power float64) error

    // SetMaxPower requests maximum power output.
    // For current controllers, this forces max phases before setting max current.
    SetMaxPower() error

    // MinPower returns the minimum non-zero charging power.
    // For current controllers: effectiveMinCurrent * Voltage * minActivePhases.
    // For power controllers: charger-reported or config minimum.
    MinPower() float64

    // MaxPower returns the maximum charging power.
    // For current controllers: effectiveMaxCurrent * Voltage * maxActivePhases, capped by vehicle/circuit.
    // For power controllers: charger-reported or config maximum.
    MaxPower() float64

    // Prepare initializes the controller with runtime dependencies.
    Prepare(log *util.Logger, clock clock.Clock)
}
```

```go
// api/api.go (new interfaces)

// PowerController provides power-based charger control in W
type PowerController interface {
    MaxPower(power float64) error
}

// PowerLimiter returns the power limits for power-controlled devices
type PowerLimiter interface {
    GetMinMaxPower() (float64, float64, error)
}
```

### Component: `DirectPowerController`

For chargers implementing `api.PowerController` (heatpumps, water heaters, SG Ready with power envelope).

```go
// core/chargercontroller/power.go

type DirectPowerController struct {
    log     *util.Logger
    charger api.Charger         // for Enable/Enabled/Status
    power   api.PowerController // for MaxPower
    limiter api.PowerLimiter    // optional, for GetMinMaxPower
    circuit api.Circuit         // optional

    offeredPower float64
    enabled      bool
    switchedAt   time.Time
    clock        clock.Clock
}
```

**Responsibilities:**
- `SetPower(watts)`: validate against circuit limits → call `charger.MaxPower(watts)` → handle enable/disable
- `SetMaxPower()`: call `SetPower(MaxPower())`
- `MinPower()`: from `PowerLimiter` if available, else config default
- `MaxPower()`: from `PowerLimiter` if available, else config default

**What it does NOT do:**
- Phase switching (no phases for power devices)
- Current rounding (no current)
- Vehicle wake-up (power devices aren't EVs)
- Resurrector handling

### Component: `CurrentController`

For standard EV wallboxes implementing `api.Charger` (+ optional `api.ChargerEx`, `api.PhaseSwitcher`).

```go
// core/chargercontroller/current.go

type CurrentController struct {
    log     *util.Logger
    clock   clock.Clock
    charger api.Charger
    circuit api.Circuit  // optional

    // current limits
    minCurrent float64
    maxCurrent float64

    // current state
    offeredCurrent float64
    chargeCurrents []float64
    chargePower    float64

    // phase state
    phases           int
    phasesConfigured int
    measuredPhases   int

    // phase switch timers
    phaseTimer     time.Time
    phasesSwitched time.Time
    enableDelay    time.Duration  // from loadpoint config
    disableDelay   time.Duration

    // charger state
    enabled         bool
    chargerSwitched time.Time

    // vehicle reference (for min/max current overrides and wake-up)
    vehicle func() api.Vehicle
}
```

**Responsibilities:**
- `SetPower(watts)`: convert to current → optimize phases → round → validate circuit → call charger → enable/disable → wake-up
- `SetMaxPower()`: force 3p (or 1p if circuit limited) → set max current
- `MinPower()`: `effectiveMinCurrent * Voltage * minActivePhases`
- `MaxPower()`: `effectiveMaxCurrent * Voltage * maxActivePhases`, capped by vehicle

**What moves here from Loadpoint:**
- `setLimit(current)` → becomes the core of `SetPower()`
- `roundedCurrent()`, `coarseCurrent()`
- `effectiveMinCurrent()`, `effectiveMaxCurrent()` (with vehicle override support)
- `effectiveCurrent()` (the hysteresis logic)
- Phase switching: `pvScalePhases()`, `scalePhases()`, `scalePhasesIfAvailable()`
- Phase state: `phases`, `measuredPhases`, `phasesConfigured`, `phaseTimer`, `phasesSwitched`
- `activePhases()`, `minActivePhases()`, `maxActivePhases()`
- `chargerSwitched` timestamp
- Vehicle `Resurrector` wake-up on error

### Phase Switching Inside `CurrentController.SetPower()`

Phase switching currently lives in `pvMaxCurrent()` (the strategy layer). It should move to the controller because it's about *how* to deliver a target power, not *what* power to deliver.

The key insight: `pvScalePhases(sitePower, minCurrent, maxCurrent)` uses `availablePower = chargePower - sitePower` to decide phases. In the new design, the controller receives `targetPower` which represents the same thing (target = current consumption + surplus).

```go
func (c *CurrentController) SetPower(power float64) error {
    // 1. Optimize phases for target power
    c.optimizePhases(power)

    // 2. Convert to current
    current := powerToCurrent(power, c.activePhases())
    current = c.roundedCurrent(current)

    // 3. Apply circuit limits
    current = c.applyCircuitLimits(current)

    // 4. Set on charger
    return c.setChargerCurrent(current)
}

func (c *CurrentController) SetMaxPower() error {
    // Force max phases, then max current
    c.forceMaxPhases()
    return c.setChargerCurrent(c.effectiveMaxCurrent())
}

func (c *CurrentController) optimizePhases(targetPower float64) {
    if !c.hasPhaseSwitching() || !c.phaseSwitchCompleted() {
        return
    }

    minPowerAtActivePhases := currentToPower(c.effectiveMinCurrent(), c.activePhases())
    maxPowerAt1p := currentToPower(c.effectiveMaxCurrent(), 1)
    minPowerAtMaxPhases := currentToPower(c.effectiveMinCurrent(), c.maxActivePhases())

    // Scale down: target power below minimum at current phases
    if targetPower < minPowerAtActivePhases && c.activePhases() > 1 {
        c.startOrCheckPhaseTimer(1, c.disableDelay)
    }
    // Scale up: target power exceeds 1p capacity AND fits minCurrent at max phases
    else if targetPower > maxPowerAt1p && targetPower >= minPowerAtMaxPhases {
        c.startOrCheckPhaseTimer(c.maxActivePhases(), c.enableDelay)
    }
    // Reset timer
    else {
        c.resetPhaseTimer()
    }
}
```

### Loadpoint Simplification

With charger control extracted, the loadpoint becomes a pure strategy engine:

**What stays in Loadpoint:**
- Mode management (off/now/minpv/pv)
- PV enable/disable timers and thresholds
- Vehicle management (identification, SOC tracking, vehicle assignments)
- Session tracking (energy, duration, sessions DB)
- Plan/planner integration (SOC plans, energy plans, repeating plans)
- Smart cost/feed-in logic
- Battery boost logic
- Status publishing to UI/websocket
- Event bus
- Settings persistence

**What the loadpoint's Update() becomes:**

```go
func (lp *Loadpoint) Update(sitePower, batteryBoostPower float64, ...) {
    // ... (unchanged: smart cost, dimmer, status, vehicle, soc, sync)

    mode := lp.GetMode()
    plannerActive := lp.plannerActive()

    switch {
    case !lp.connected():
        err = lp.controller.SetPower(0)

    case mode == api.ModeOff:
        var power float64
        if welcomeCharge { power = lp.controller.MinPower() }
        err = lp.controller.SetPower(power)

    case lp.minSocNotReached() || plannerActive:
        err = lp.controller.SetMaxPower()
        lp.elapsePVTimer()

    case lp.LimitEnergyReached(), lp.LimitSocReached():
        err = lp.disableUnlessClimater()

    case mode == api.ModeNow:
        err = lp.controller.SetMaxPower()

    case mode == api.ModeMinPV || mode == api.ModePV:
        if smartCostActive {
            err = lp.controller.SetMaxPower()
            lp.elapsePVTimer()
            break
        }
        if smartFeedInPriorityActive {
            power := float64(0)
            if mode == api.ModeMinPV { power = lp.controller.MinPower() }
            err = lp.controller.SetPower(power)
            lp.elapsePVTimer()
            break
        }

        targetPower := lp.pvMaxPower(mode, sitePower, batteryBoostPower, batteryBuffered, batteryStart)
        if targetPower == 0 && lp.vehicleClimateActive() { targetPower = lp.controller.MinPower() }
        if targetPower == 0 && welcomeCharge { targetPower = lp.controller.MinPower(); lp.resetPVTimer() }
        err = lp.controller.SetPower(targetPower)
    }
}
```

**pvMaxPower() replaces pvMaxCurrent():**

```go
func (lp *Loadpoint) pvMaxPower(mode api.ChargeMode, sitePower, batteryBoostPower float64, batteryBuffered, batteryStart bool) float64 {
    minPower := lp.controller.MinPower()
    maxPower := lp.controller.MaxPower()

    sitePower -= lp.boostPower(batteryBoostPower)

    // Target power = current consumption + available surplus
    targetPower := max(lp.chargePower + (-sitePower), 0)

    // MinPV / battery buffered floor
    if battery := batteryStart || batteryBuffered && lp.charging(); (mode == api.ModeMinPV || battery) && targetPower < minPower {
        return minPower
    }

    // PV disable timer
    if mode == api.ModePV && lp.enabled && targetPower < minPower {
        if sitePower >= lp.Disable.Threshold {
            // ... timer logic (identical to current, thresholds already in W)
            return 0  // after delay
        }
        return minPower
    }

    // PV enable timer
    if mode == api.ModePV && !lp.enabled {
        if (lp.Enable.Threshold == 0 && targetPower >= minPower) ||
           (lp.Enable.Threshold != 0 && sitePower <= lp.Enable.Threshold) {
            // ... timer logic
            return minPower  // after delay
        }
        return 0
    }

    return min(targetPower, maxPower)
}
```

**Key simplification:** No phase awareness. No `effectiveCurrent()` with its Zoe hysteresis. No `scaledTo == 3` adjustment. No `IntegratedDevice` special case. Just `chargePower + surplus = target`. The controller handles everything else.

### Charger Migration

Each power-native charger adds `api.PowerController`, keeps `MaxCurrentMillis` for backward compat, removes `loadpoint.Controller` dependency:

**charger/heatpump.go:**
```go
var _ api.PowerController = (*Heatpump)(nil)

func (wb *Heatpump) MaxPower(power float64) error {
    return wb.setMaxPower(int64(power))
}

// Remove: lp loadpoint.API field
// Remove: LoadpointControl method
// Keep: MaxCurrentMillis for backward compat (uses fixed phases=1)
```

**charger/mypv.go, charger/ego.go, charger/sgready.go:** Same pattern.

### `effectiveCurrent()` in Power Terms

The current `effectiveCurrent()` adds +2A hysteresis to handle vehicles like the Renault Zoe that don't charge at the full offered current. In the power approach, this becomes a controller concern:

In `CurrentController`, when the loadpoint asks "what is the current charge power?", the controller can provide a corrected value:

```go
// EffectiveChargePower returns the charge power adjusted for soft vehicles
func (c *CurrentController) EffectiveChargePower() float64 {
    if c.chargeCurrents != nil {
        cur := max(c.chargeCurrents[0], c.chargeCurrents[1], c.chargeCurrents[2])
        effectiveCur := min(cur+2.0, c.offeredCurrent)
        return effectiveCur * float64(c.activePhases()) * Voltage
    }
    return c.chargePower
}
```

The loadpoint's `pvMaxPower` would use `lp.controller.EffectiveChargePower()` instead of `lp.chargePower` to get the same behavior.

Alternatively, this could be added to the `Controller` interface:
```go
type Controller interface {
    // ...
    EffectiveChargePower() float64  // charge power adjusted for controller-specific behavior
}
```

For `DirectPowerController`, this simply returns `chargePower` (no adjustment needed for power devices). This also replaces the `IntegratedDevice` special case that currently exists in `pvMaxCurrent`.

## Migration Path

### Phase 1: Define interfaces and create controller package

**New files:**
- `core/chargercontroller/controller.go` — `Controller` interface
- `core/chargercontroller/power.go` — `DirectPowerController`
- `core/chargercontroller/current.go` — `CurrentController`

**Modified files:**
- `api/api.go` — add `PowerController`, `PowerLimiter` interfaces

**Testing:** Unit tests for both controller implementations in isolation.

### Phase 2: Create `CurrentController` by extracting from Loadpoint

Move these methods from `core/loadpoint.go` and `core/loadpoint_phases.go` into `CurrentController`:

| From Loadpoint | To CurrentController |
|----------------|---------------------|
| `setLimit(current)` | `setChargerCurrent(current)` (private, called by `SetPower`) |
| `roundedCurrent()` / `coarseCurrent()` | same names, private |
| `effectiveMinCurrent()` / `effectiveMaxCurrent()` | same, private |
| `effectiveCurrent()` | `EffectiveChargePower()` (returns watts) |
| `pvScalePhases()` | `optimizePhases(targetPower)` |
| `scalePhases()` / `scalePhasesIfAvailable()` | private helpers |
| `activePhases()` / `minActivePhases()` / `maxActivePhases()` | private, with public `ActivePhases()` |
| `hasPhaseSwitching()` | private |
| Phase state fields: `phases`, `measuredPhases`, `phasesConfigured`, `phaseTimer`, `phasesSwitched` | controller fields |
| `offeredCurrent` | controller field |
| `chargerSwitched` | controller field |

**Critical:** The loadpoint still needs to read some controller state for publishing to UI (active phases, offered current, phase timer). The controller should expose these via read-only methods.

**Testing:** Existing `setLimit` tests adapted for `SetPower`. Phase switching tests stay similar.

### Phase 3: Create `DirectPowerController`

Simpler implementation. New code.

**Testing:** New unit tests.

### Phase 4: Wire controllers into Loadpoint

**Modified files:**
- `core/loadpoint.go`:
  - Add `controller chargercontroller.Controller` field
  - In `Prepare()`: create `DirectPowerController` or `CurrentController` based on charger type
  - Replace `pvMaxCurrent()` with `pvMaxPower()`
  - Replace all `setLimit()` calls with `controller.SetPower()`
  - Replace `fastCharging()` with `controller.SetMaxPower()`
  - Replace `disableUnlessClimater()` to use `controller.SetPower()`
  - Update `evChargeCurrentWrappedMeterHandler` to use controller
  - Remove extracted fields and methods
- `core/loadpoint_phases.go`: Remove methods moved to controller (keep `ActivePhases()` as delegation to controller)
- `core/loadpoint_effective.go`: Simplify — `EffectiveMinPower()` and `EffectiveMaxPower()` delegate to controller

**Testing:** Full integration tests with mock controllers.

### Phase 5: Migrate chargers

**Modified files:**
- `charger/heatpump.go` — add `PowerController`, remove `loadpoint.Controller`
- `charger/mypv.go` — add `PowerController`, remove `loadpoint.Controller`
- `charger/ego.go` — add `PowerController`, remove `loadpoint.Controller`
- `charger/sgready.go` — add `PowerController`, remove `loadpoint.Controller`

**Testing:** Charger unit tests, integration tests.

### Phase 6: Clean up

- Remove `loadpoint.Controller` interface if no longer used
- Remove `powerToCurrent`/`currentToPower` from `core/helper.go` (move to controller package as private)
- Regenerate mocks: `go generate ./api/... ./core/...`
- Update WebSocket publishes for new state locations
- Update frontend if needed (`powerControlled` flag, `offeredPower` field)

## State Publishing

The controller needs to publish state for the UI. Options:

**Option A: Controller publishes directly** via a `publish func(key string, val interface{})` callback.

**Option B: Loadpoint reads controller state** each cycle and publishes. Cleaner separation but more boilerplate.

**Recommendation:** Option A for simplicity. The controller receives a `publish` function in `Prepare()`.

## What Stays Unchanged

- `core/site.go` — no changes to power distribution
- `api.Charger` interface — backward compatible
- Circuit validation — reused by both controllers
- Planner — works with `EffectiveMaxPower()` which delegates to controller
- Frontend — mostly unchanged; `powerControlled` + `offeredPower` are additive
- All existing EV charger implementations — they continue implementing `api.Charger`; the `CurrentController` wraps them without any changes

## Risk Analysis

| Risk | Mitigation |
|------|-----------|
| Phase switching behavior changes subtly | Comprehensive test coverage of phase switching with clock mocking |
| Published state (WebSocket) keys change | Keep same keys, source from controller |
| `effectiveCurrent` hysteresis regression | Test with Zoe-like scenarios |
| Loadpoint tests break | Tests are the main work; extract test helpers |
| Third-party charger plugins break | Keep `api.Charger` interface unchanged; `loadpoint.Controller` stays temporarily |

## Summary of Complexity Reduction

| Before | After |
|--------|-------|
| Loadpoint manages current, phases, and power | Loadpoint manages power only |
| `pvMaxCurrent()` — 120 lines with phase switching | `pvMaxPower()` — ~50 lines, no phase awareness |
| `setLimit()` — 90 lines with circuit+enable+wake-up | `controller.SetPower()` — same logic, but in the right place |
| `IntegratedDevice` special cases scattered | `DirectPowerController` handles power devices cleanly |
| Chargers need `loadpoint.Controller` to query phases | Chargers implement `PowerController`, no loadpoint coupling |
| Power→current→power round-trip | Direct power path, no conversion loss |
