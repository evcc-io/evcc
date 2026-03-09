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
controller.SetOfferedPower(watts)
        ↓
┌─────────────────────────┬──────────────────────────────────┐
│ PowerController         │ CurrentController                │
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
// The loadpoint calls SetOfferedPower() with a target in watts;
// the controller handles the specifics of how to deliver that power.
type Controller interface {
    // SetOfferedPower sets the target charging power in watts.
    // 0 or below MinPower means disable.
    SetOfferedPower(power float64) error

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

// PowerController provides power-based charger control in W.
// Chargers that natively accept power targets implement this interface.
type PowerController interface {
    MaxPower(power float64) error
}

// PowerLimiter returns the power limits for power-controlled devices
type PowerLimiter interface {
    GetMinMaxPower() (float64, float64, error)
}
```

### Component: `PowerController`

For chargers implementing `api.PowerController` (heatpumps, water heaters, SG Ready with power envelope).

```go
// core/chargercontroller/power.go

type PowerController struct {
    log     *util.Logger
    charger api.Charger      // for Enable/Enabled/Status
    power   api.PowerController // for MaxPower
    limiter api.PowerLimiter // optional, for GetMinMaxPower
    circuit api.Circuit      // optional

    offeredPower float64
    enabled      bool
    switchedAt   time.Time
    clock        clock.Clock
}
```

**Responsibilities:**
- `SetOfferedPower(watts)`: validate against circuit limits → call `charger.MaxPower(watts)` → handle enable/disable
- `SetMaxPower()`: call `SetOfferedPower(MaxPower())`
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
- `SetOfferedPower(watts)`: convert to current → optimize phases → round → validate circuit → call charger → enable/disable → wake-up
- `SetMaxPower()`: force 3p (or 1p if circuit limited) → set max current
- `MinPower()`: `effectiveMinCurrent * Voltage * minActivePhases`
- `MaxPower()`: `effectiveMaxCurrent * Voltage * maxActivePhases`, capped by vehicle

**What moves here from Loadpoint:**
- `setLimit(current)` → becomes the core of `SetOfferedPower()`
- `roundedCurrent()`, `coarseCurrent()`
- `effectiveMinCurrent()`, `effectiveMaxCurrent()` (with vehicle override support)
- `effectiveCurrent()` (the hysteresis logic)
- Phase switching: `pvScalePhases()`, `scalePhases()`, `scalePhasesIfAvailable()`
- Phase state: `phases`, `measuredPhases`, `phasesConfigured`, `phaseTimer`, `phasesSwitched`
- `activePhases()`, `minActivePhases()`, `maxActivePhases()`
- `chargerSwitched` timestamp
- Vehicle `Resurrector` wake-up on error

### Phase Switching Inside `CurrentController.SetOfferedPower()`

Phase switching currently lives in `pvMaxCurrent()` (the strategy layer). It should move to the controller because it's about *how* to deliver a target power, not *what* power to deliver.

The key insight: `pvScalePhases(sitePower, minCurrent, maxCurrent)` uses `availablePower = chargePower - sitePower` to decide phases. In the new design, the controller receives `targetPower` which represents the same thing (target = current consumption + surplus).

```go
func (c *CurrentController) SetOfferedPower(power float64) error {
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
        err = lp.controller.SetOfferedPower(0)

    case mode == api.ModeOff:
        var power float64
        if welcomeCharge { power = lp.controller.MinPower() }
        err = lp.controller.SetOfferedPower(power)

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
            err = lp.controller.SetOfferedPower(power)
            lp.elapsePVTimer()
            break
        }

        targetPower := lp.pvMaxPower(mode, sitePower, batteryBoostPower, batteryBuffered, batteryStart)
        if targetPower == 0 && lp.vehicleClimateActive() { targetPower = lp.controller.MinPower() }
        if targetPower == 0 && welcomeCharge { targetPower = lp.controller.MinPower(); lp.resetPVTimer() }
        err = lp.controller.SetOfferedPower(targetPower)
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

For the `PowerController`, this simply returns `chargePower` (no adjustment needed for power devices). This also replaces the `IntegratedDevice` special case that currently exists in `pvMaxCurrent`.

## Migration Path

### Phase 1: Define interfaces and create controller package ✅ DONE

**Created files:**
- `core/chargercontroller/controller.go` — `Controller` interface
- `core/chargercontroller/power.go` — `PowerController` struct
- `core/loadpoint_controller.go` — `currentControllerAdapter` (transitional thin wrapper)

**Modified files:**
- `api/api.go` — added `PowerController`, `PowerLimiter` interfaces

### Phase 2: Wire controllers into Loadpoint ✅ DONE

**Modified files:**
- `core/loadpoint.go`:
  - Added `controller chargercontroller.Controller` field
  - In `Prepare()`: creates `PowerController` or `currentControllerAdapter` based on charger type
  - `Update()` strategy switch uses `controller.SetOfferedPower()` / `controller.SetMaxPower()` / `controller.MinPower()`
  - `disableUnlessClimater()` uses controller
  - `evChargeCurrentWrappedMeterHandler()` uses `controller.EffectiveChargePower()`
  - `UpdateChargePowerAndCurrents()` feeds measured power to `PowerController`
- `core/loadpoint_effective.go`: `EffectiveMinPower()` and `EffectiveMaxPower()` delegate to controller

### Phase 3: Migrate chargers ✅ DONE

**Modified files:**
- `charger/heatpump.go` — added `api.PowerController` (`MaxPower`)
- `charger/mypv.go` — added `api.PowerController` (`MaxPower`)
- `charger/ego.go` — added `api.PowerController` (`MaxPower`)
- `charger/sgready.go` — added `api.PowerController` (`MaxPower`)

Note: `loadpoint.Controller` dependency kept for backward compat. Remove after Phase 4.

---

### Phase 4: Extract `CurrentController` from Loadpoint — THE BIG REFACTOR

The `currentControllerAdapter` in `core/loadpoint_controller.go` is a thin wrapper that delegates everything back to the loadpoint. This means all current/phase logic still lives in `loadpoint.go`. The goal of Phase 4 is to extract this logic into a standalone `CurrentController` in `core/chargercontroller/current.go`, making the loadpoint operate exclusively in watts.

#### 4.1 The Problem with the Current State

The `currentControllerAdapter` just delegates:
```go
func (a *currentControllerAdapter) SetOfferedPower(power float64) error {
    current := powerToCurrent(power, a.lp.ActivePhases())
    return a.lp.setLimit(current)  // ALL logic still in loadpoint
}
```

The loadpoint still owns ~400 lines of current-specific logic:

| Method/Field | Lines | What it does |
|---|---|---|
| `setLimit(current)` | ~90 | Round, circuit validate, set charger current, enable/disable, vehicle wake-up |
| `fastCharging()` | ~20 | Force 3p + max current |
| `pvMaxCurrent()` | ~120 | Convert surplus W→A, phase switching, PV timers |
| `pvScalePhases()` | ~80 | Phase switching decisions with timers |
| `scalePhases()`/`scalePhasesIfAvailable()` | ~25 | Execute phase switch on charger |
| `effectiveCurrent()` | ~12 | Current with Zoe hysteresis |
| `effectiveMinCurrent()`/`effectiveMaxCurrent()` | ~45 | Min/max from config+vehicle+charger |
| `roundedCurrent()`/`coarseCurrent()` | ~10 | Integer rounding |
| Phase fields | — | `phases`, `measuredPhases`, `phasesConfigured`, `phaseTimer`, `phasesSwitched`, `offeredCurrent`, `chargeCurrents`, `chargerSwitched` |
| Phase helpers | ~60 | `activePhases()`, `minActivePhases()`, `maxActivePhases()`, `hasPhaseSwitching()`, `phaseSwitchCompleted()`, `chargerUpdateCompleted()`, `phasesFromChargeCurrents()` |

#### 4.2 Target Architecture

```
core/chargercontroller/
├── controller.go          # Controller interface (exists)
├── power.go               # PowerController (exists)
└── current.go             # CurrentController (NEW - extracted from loadpoint)
```

The `CurrentController` is a standalone struct that:
1. Implements `chargercontroller.Controller`
2. Owns ALL current, phase, and charger enable/disable state
3. Receives power targets in watts, handles conversion internally
4. Exposes read-only accessors for state the loadpoint needs to publish

#### 4.3 `Host` Interface — Controller → Loadpoint Dependency

The `CurrentController` needs dynamic, vehicle-dependent state from the loadpoint. Rather than copying vehicle state into the controller (via `UpdateVehicle`), the controller calls back through a minimal `Host` interface that the loadpoint implements:

```go
// core/chargercontroller/host.go

// Host provides the controller with access to vehicle-dependent state
// and loadpoint callbacks. The loadpoint implements this interface.
type Host interface {
    // GetVehicle returns the currently active vehicle, or nil.
    // The controller queries vehicle limits, phases, and features directly.
    GetVehicle() api.Vehicle

    // WakeUpVehicle attempts to wake a sleeping vehicle.
    // Returns nil if no vehicle is connected or it has no Resurrector.
    WakeUpVehicle() error

    // Charging returns true if the charger status is C (actively charging).
    Charging() bool

    // StartWakeUpTimer starts the vehicle wake-up timeout.
    StartWakeUpTimer()

    // StopWakeUpTimer stops the vehicle wake-up timeout.
    StopWakeUpTimer()
}
```

**Why a Host interface instead of `UpdateVehicle()`:**
- Vehicle state (limits, phases, features) can change at any time — not just on vehicle assignment
- The controller always reads the latest values via `host.GetVehicle()`, no stale state
- No synchronization burden on the loadpoint to call `UpdateVehicle` at the right times
- The loadpoint already has `GetVehicle()` as a public method; the interface just formalizes the contract
- Testable: mock the Host interface in controller tests

**Loadpoint implementation:**

```go
// core/loadpoint_host.go

// Verify Loadpoint implements chargercontroller.Host
var _ chargercontroller.Host = (*Loadpoint)(nil)

func (lp *Loadpoint) WakeUpVehicle() error {
    v := lp.GetVehicle()
    if v == nil {
        return nil
    }
    if r, ok := v.(api.Resurrector); ok {
        return r.WakeUp()
    }
    return nil
}

func (lp *Loadpoint) Charging() bool {
    return lp.charging()
}

func (lp *Loadpoint) StartWakeUpTimer() {
    lp.startWakeUpTimer()
}

func (lp *Loadpoint) StopWakeUpTimer() {
    lp.stopWakeUpTimer()
}
```

Note: `GetVehicle()` already exists on `Loadpoint` — no new method needed.

#### 4.3b `CurrentController` Struct Design

```go
// core/chargercontroller/current.go

type CurrentController struct {
    log   *util.Logger
    clock clock.Clock
    host  Host // loadpoint callbacks (vehicle limits, wake-up, charging state)

    // charger interfaces (set at creation, immutable)
    charger        api.Charger
    chargerEx      api.ChargerEx      // optional: milliamp precision
    phaseSwitcher  api.PhaseSwitcher  // optional: 1p/3p switching
    currentLimiter api.CurrentLimiter // optional: charger min/max current
    circuit        api.Circuit        // optional: load management

    // configuration
    minCurrent       float64 // PV mode start current
    maxCurrent       float64 // max allowed current
    phasesConfigured int     // 0=auto, 1, 3

    // charger control state
    offeredCurrent  float64
    enabled         bool
    chargerSwitched time.Time

    // phase state
    phases         int       // charger enabled phases
    measuredPhases int       // physically measured phases
    phaseTimer     time.Time // 1p3p switch timer
    phasesSwitched time.Time // last phase switch timestamp

    // measurement state (fed from loadpoint's meter reads)
    chargePower    float64   // measured charge power
    chargeCurrents []float64 // measured per-phase currents

    // timing configuration (from loadpoint enable/disable delay config)
    enableDelay  time.Duration
    disableDelay time.Duration

    // callbacks
    setEnabled func(bool)              // update loadpoint enabled state
    publish    func(key string, val any) // publish state to UI
}
```

**Key difference from `UpdateVehicle` approach**: No vehicle-related fields on the controller. Instead, methods call `c.host.GetVehicle()` and query the vehicle directly, always getting current values. Vehicle wake-up calls `c.host.WakeUpVehicle()`.

```go
func (c *CurrentController) effectiveMinCurrent() float64 {
    var vehicleMin, chargerMin float64
    if v := c.host.GetVehicle(); v != nil {
        if res, ok := v.OnIdentified().GetMinCurrent(); ok {
            vehicleMin = res
        }
    }
    if c.currentLimiter != nil {
        if res, _, err := c.currentLimiter.GetMinMaxCurrent(); err == nil {
            chargerMin = res
        }
    }
    switch {
    case max(vehicleMin, chargerMin) == 0:
        return c.minCurrent
    case chargerMin > 0:
        return max(vehicleMin, chargerMin)
    default:
        return max(vehicleMin, c.minCurrent)
    }
}

func (c *CurrentController) coarseCurrent() bool {
    if c.chargerEx == nil {
        return true
    }
    if v := c.host.GetVehicle(); v != nil {
        return slices.Contains(v.Features(), api.CoarseCurrent)
    }
    return false
}

func (c *CurrentController) getVehiclePhases() int {
    if v := c.host.GetVehicle(); v != nil {
        return v.Phases()
    }
    return 0
}
```

#### 4.4 Method Extraction Map

| Loadpoint Method | → CurrentController Method | Notes |
|---|---|---|
| `setLimit(current float64) error` | `setChargerCurrent(current float64) error` | Private. Core of `SetOfferedPower`. |
| `fastCharging() error` | `SetMaxPower() error` | Forces max phases, then max current. |
| `roundedCurrent(current) float64` | `roundedCurrent(current) float64` | Private. |
| `coarseCurrent() bool` | `coarseCurrent() bool` | Private. Uses `chargerEx` and `coarseCurrentMode`. |
| `effectiveMinCurrent() float64` | `effectiveMinCurrent() float64` | Private. Uses `minCurrent`, `vehicleMinCurrent`, `currentLimiter`. |
| `effectiveMaxCurrent() float64` | `effectiveMaxCurrent() float64` | Private. Uses `maxCurrent`, `vehicleMaxCurrent`, `currentLimiter`. |
| `effectiveCurrent() float64` | Used internally by `EffectiveChargePower()` | Zoe +2A hysteresis logic. |
| `pvScalePhases(sitePower, min, max)` | `optimizePhases(targetPower float64)` | Called inside `SetOfferedPower`. Rewritten to work with target power instead of sitePower+current. |
| `scalePhases(phases int) error` | `scalePhases(phases int) error` | Private. Calls `phaseSwitcher.Phases1p3p`. |
| `scalePhasesIfAvailable(phases)` | `scalePhasesIfAvailable(phases)` | Private. |
| `activePhases() int` | `activePhases() int` | Private. Public `ActivePhases()` for loadpoint reads. |
| `minActivePhases() int` | `minActivePhases() int` | Private. |
| `maxActivePhases() int` | `maxActivePhases() int` | Private. |
| `hasPhaseSwitching() bool` | `hasPhaseSwitching() bool` | Private. Checks `phaseSwitcher != nil`. |
| `phaseSwitchCompleted() bool` | `phaseSwitchCompleted() bool` | Private. |
| `chargerUpdateCompleted() bool` | `chargerUpdateCompleted() bool` | Private. |
| `resetPhaseTimer()` | `resetPhaseTimer()` | Private. |
| `publishTimer()` | Calls `publish` callback | Reuse same keys. |
| `phasesFromChargeCurrents()` | `updateMeasuredPhases()` | Called when charge currents are updated. |
| `effectiveMaxPower() float64` | `MaxPower() float64` | Combines `effectiveMaxCurrent * Voltage * maxActivePhases`, capped by vehicle max power via `host.GetVehicle()`. |

#### 4.5 `SetOfferedPower` Implementation

The key method. Replaces `setLimit(current)` + phase-switching that was embedded in `pvMaxCurrent()`:

```go
func (c *CurrentController) SetOfferedPower(power float64) error {
    // 1. Phase optimization: decide whether to switch 1p↔3p
    //    This replaces pvScalePhases which was called from pvMaxCurrent.
    //    Uses target power to decide, not sitePower + deltas.
    if c.hasPhaseSwitching() && c.phaseSwitchCompleted() {
        c.optimizePhases(power)
    }

    // 2. Convert power to current at active phases
    activePhases := c.activePhases()
    current := powerToCurrent(power, activePhases)
    current = c.roundedCurrent(current)

    // 3. Apply circuit limits (both current and power based)
    if c.circuit != nil {
        var actualCurrent float64
        if c.chargeCurrents != nil {
            actualCurrent = max(c.chargeCurrents[0], c.chargeCurrents[1], c.chargeCurrents[2])
        } else if c.host.Charging() {
            actualCurrent = c.offeredCurrent
        }

        currentLimit := c.circuit.ValidateCurrent(actualCurrent, current)
        powerLimit := c.circuit.ValidatePower(c.chargePower, currentToPower(current, activePhases))
        currentLimitViaPower := powerToCurrent(powerLimit, activePhases)
        current = c.roundedCurrent(min(currentLimit, currentLimitViaPower))
    }

    // 4. Validate min/max
    effMinCurrent := c.effectiveMinCurrent()
    if effMaxCurrent := c.effectiveMaxCurrent(); effMinCurrent > effMaxCurrent {
        return fmt.Errorf("invalid config: min current %.3gA exceeds max current %.3gA", effMinCurrent, effMaxCurrent)
    }

    // 5. Set current on charger
    if current != c.offeredCurrent && current >= effMinCurrent {
        if err := c.setChargerCurrent(current); err != nil {
            return err
        }
    }

    // 6. Enable/disable
    if enabled := current >= effMinCurrent; enabled != c.enabled {
        if err := c.charger.Enable(enabled); err != nil {
            // vehicle wake-up on ErrAsleep
            if enabled && errors.Is(err, api.ErrAsleep) {
                if wakeErr := c.host.WakeUpVehicle(); wakeErr != nil {
                    return fmt.Errorf("wake-up vehicle: %w", wakeErr)
                }
            }
            return fmt.Errorf("charger %s: %w", enabledStatus[enabled], err)
        }

        c.enabled = enabled
        c.setEnabled(enabled)
        c.chargerSwitched = c.clock.Now()

        if !enabled {
            c.offeredCurrent = 0
        }
        c.publish(keys.ChargeCurrent, c.offeredCurrent)

        if enabled {
            c.host.StartWakeUpTimer()
        } else {
            c.host.StopWakeUpTimer()
        }
    }

    return nil
}
```

#### 4.6 Phase Optimization (`optimizePhases`)

Replaces `pvScalePhases()`. The key difference: it operates on target power, not on sitePower + effective current deltas. The loadpoint's `pvMaxPower()` has already calculated the target; the controller just decides the optimal phase count to deliver it.

```go
func (c *CurrentController) optimizePhases(targetPower float64) {
    activePhases := c.activePhases()
    effMinCurrent := c.effectiveMinCurrent()
    effMaxCurrent := c.effectiveMaxCurrent()

    minPowerAtActive := currentToPower(effMinCurrent, activePhases)
    maxPowerAt1p := currentToPower(effMaxCurrent, 1)

    scalable := (targetPower < minPowerAtActive || !c.enabled) &&
                activePhases > 1 && c.phasesConfigured < 3

    // Scale down: target power below minimum at current phase count
    if scalable {
        if !c.host.Charging() {
            c.phaseTimer = elapsed // scale immediately if not charging
        }
        if c.phaseTimer.IsZero() {
            c.phaseTimer = c.clock.Now()
        }
        c.publishTimer(phaseTimer, c.disableDelay, phaseScale1p)

        if c.clock.Since(c.phaseTimer) >= c.disableDelay {
            _ = c.scalePhases(1)
            c.phaseTimer = time.Time{}
        }
        return
    }

    // Scale up: target power exceeds 1p max AND fits minimum at max phases
    maxPhases := c.maxActivePhases()
    minPowerAtMax := currentToPower(effMinCurrent, maxPhases)

    if activePhases == 1 && targetPower > maxPowerAt1p && targetPower >= minPowerAtMax {
        if c.phaseTimer.IsZero() {
            c.phaseTimer = c.clock.Now()
        }
        c.publishTimer(phaseTimer, c.enableDelay, phaseScale3p)

        if c.clock.Since(c.phaseTimer) >= c.enableDelay {
            _ = c.scalePhases(maxPhases)
            c.phaseTimer = time.Time{}
        }
        return
    }

    // No scaling needed — reset timer
    c.resetPhaseTimer()
}
```

#### 4.7 Read-Only Accessors for Loadpoint

The loadpoint needs to read controller state for UI publishing, SOC estimation, and API responses:

```go
// Read-only accessors exposed to the loadpoint
func (c *CurrentController) ActivePhases() int          // for UI, SOC estimation
func (c *CurrentController) GetPhases() int             // enabled phases
func (c *CurrentController) GetPhasesConfigured() int   // for API
func (c *CurrentController) GetMeasuredPhases() int     // for diagnostics
func (c *CurrentController) GetOfferedCurrent() float64 // for UI, session tracking
func (c *CurrentController) HasPhaseSwitching() bool    // for UI
func (c *CurrentController) IsChargerUpdateCompleted() bool
func (c *CurrentController) IsPhaseSwitchCompleted() bool

// Setters called by the loadpoint
func (c *CurrentController) SetPhasesConfigured(phases int) error  // from API
func (c *CurrentController) SetMinCurrent(current float64)         // from API
func (c *CurrentController) SetMaxCurrent(current float64)         // from API
func (c *CurrentController) UpdateChargePower(power float64)       // from meter
func (c *CurrentController) UpdateChargeCurrents(currents []float64) // from meter
func (c *CurrentController) ResetMeasuredPhases()                  // on disconnect/phase switch
```

#### 4.8 `pvMaxPower()` — Replaces `pvMaxCurrent()` in Loadpoint

With current/phase logic extracted, `pvMaxCurrent()` (~120 lines) becomes `pvMaxPower()` (~50 lines):

```go
func (lp *Loadpoint) pvMaxPower(mode api.ChargeMode, sitePower, batteryBoostPower float64,
    batteryBuffered, batteryStart bool) float64 {

    minPower := lp.controller.MinPower()
    maxPower := lp.controller.MaxPower()

    // push demand to drain battery
    sitePower -= lp.boostPower(batteryBoostPower)

    // target power = current consumption + available surplus
    effectiveChargePower := lp.controller.EffectiveChargePower()
    targetPower := max(effectiveChargePower-sitePower, 0)

    // MinPV / battery-buffered floor
    if battery := batteryStart || batteryBuffered && lp.charging();
        (mode == api.ModeMinPV || battery) && targetPower < minPower {
        return minPower
    }

    lp.log.DEBUG.Printf("pv charge power: %.0fW = %.0fW + %.0fW (%.0fW site)",
        targetPower, effectiveChargePower, effectiveChargePower-targetPower, sitePower)

    // PV disable timer (thresholds are already in watts)
    if mode == api.ModePV && lp.enabled && targetPower < minPower {
        if sitePower >= lp.Disable.Threshold {
            // ... same timer logic as current pvMaxCurrent, just returning 0/minPower instead of 0/minCurrent
            if elapsed := lp.clock.Since(lp.pvTimer); elapsed >= lp.GetDisableDelay() {
                lp.resetPVTimer()
                return 0
            }
        } else {
            lp.resetPVTimer("disable")
        }
        return minPower
    }

    // PV enable timer
    if mode == api.ModePV && !lp.enabled {
        if (lp.Enable.Threshold == 0 && targetPower >= minPower) ||
           (lp.Enable.Threshold != 0 && sitePower <= lp.Enable.Threshold) {
            if elapsed := lp.clock.Since(lp.pvTimer); elapsed >= lp.GetEnableDelay() {
                lp.resetPVTimer()
                return minPower
            }
        } else {
            lp.resetPVTimer("enable")
        }
        return 0
    }

    lp.resetPVTimer()
    return min(targetPower, maxPower)
}
```

**Key simplifications vs `pvMaxCurrent()`:**
- No `effectiveCurrent()` with its Zoe hysteresis — controller provides `EffectiveChargePower()` which already handles this
- No `IntegratedDevice` special case — `EffectiveChargePower()` handles it
- No `pvScalePhases()` call — phase switching moved to `CurrentController.SetOfferedPower()`
- No `scaledTo == 3` adjustment — not needed
- No `powerToCurrent`/`currentToPower` conversions — everything is watts
- No `activePhases` local variable — not needed
- ~50 lines vs ~120 lines

#### 4.9 Loadpoint Cleanup — What Gets Removed

After extraction, remove from `core/loadpoint.go`:
- **Fields**: `offeredCurrent`, `chargerSwitched`, `phasesSwitched`, `phases`, `measuredPhases`, `phasesConfigured`, `chargeCurrents`, `phaseTimer`
- **Methods**: `setLimit()`, `fastCharging()`, `effectiveCurrent()`, `roundedCurrent()`, `coarseCurrent()`, `pvMaxCurrent()`, `pvScalePhases()`, `scalePhases()`, `scalePhasesIfAvailable()`, `scalePhasesRequired()`

Remove from `core/loadpoint_phases.go`:
- **Methods**: `activePhases()`, `minActivePhases()`, `maxActivePhases()`, `hasPhaseSwitching()`, `phaseSwitchCompleted()`, `chargerUpdateCompleted()`, `resetPhaseTimer()`, `phasesFromChargeCurrents()`
- Keep `ActivePhases()` as delegation: `func (lp *Loadpoint) ActivePhases() int { return lp.controller.ActivePhases() }`
- Keep `GetPhases()`, `GetPhasesConfigured()`, `SetPhasesConfigured()` as delegation to controller

Remove from `core/loadpoint_effective.go`:
- **Methods**: `effectiveMinCurrent()`, `effectiveMaxCurrent()`
- Simplify `EffectiveMinPower()` → `controller.MinPower()`
- Simplify `EffectiveMaxPower()` → `controller.MaxPower()` with circuit cap
- Keep publishing `EffectiveMinCurrent`/`EffectiveMaxCurrent` by reading from controller for UI compat

Remove from `core/loadpoint_controller.go`:
- **Entire file** — `currentControllerAdapter` no longer needed

#### 4.10 Incremental Sub-Steps

The extraction is too large for a single commit. Break it into sub-steps that compile and pass tests at each point:

**Step 4A: Create `CurrentController` skeleton + `Host` interface**
- Create `core/chargercontroller/host.go` with the `Host` interface
- Create `core/chargercontroller/current.go` with struct, constructor, and stubbed methods
- Create `core/loadpoint_host.go` — loadpoint implements `Host` interface
- The constructor accepts `Host`, charger, circuit, clock, logger, config, publish callback
- All methods initially delegate back to equivalent loadpoint methods via the `Host` interface
- Tests: builds, all existing tests pass (no behavior change)

**Step 4B: Move `setLimit` → `setChargerCurrent`**
- Move the current-setting logic (charger.MaxCurrent, circuit validation, enable/disable, vehicle wake-up) into `CurrentController`
- `currentControllerAdapter.SetOfferedPower` → `CurrentController.SetOfferedPower`
- Remove `setLimit` from loadpoint
- Tests: `TestSetLimit*` adapted to test `CurrentController.SetOfferedPower`

**Step 4C: Move phase state and switching**
- Move `phases`, `measuredPhases`, `phasesConfigured`, `phaseTimer`, `phasesSwitched` fields
- Move `scalePhases`, `scalePhasesIfAvailable`, `hasPhaseSwitching`, `activePhases`, `minActivePhases`, `maxActivePhases`
- Move `phasesFromChargeCurrents` → `updateMeasuredPhases`
- Loadpoint delegates `ActivePhases()`, `GetPhases()`, etc. to controller
- Tests: `TestPvScalePhases*`, `TestFastChargingCircuitBasedPhaseScaling` adapted

**Step 4D: Move `fastCharging` → `SetMaxPower`**
- Move phase-forcing + max current logic
- Tests: existing `fastCharging` tests adapted

**Step 4E: Move effective current calculation**
- Move `effectiveMinCurrent`, `effectiveMaxCurrent`, `effectiveCurrent`, `roundedCurrent`, `coarseCurrent`
- Move `offeredCurrent`, `chargeCurrents`
- Controller receives vehicle overrides via `UpdateVehicle()`
- Tests: `TestEffectiveMinMaxCurrent` adapted

**Step 4F: Integrate `optimizePhases` into `SetOfferedPower`**
- Replace external `pvScalePhases()` call (from `pvMaxCurrent`) with internal phase optimization
- This is the step where phase switching moves from strategy to control layer
- Tests: new tests for power-based phase optimization

**Step 4G: Replace `pvMaxCurrent` with `pvMaxPower`**
- Create `pvMaxPower()` that works entirely in watts
- Update `Update()` PV path to use `pvMaxPower()` → `controller.SetOfferedPower()`
- Remove `pvMaxCurrent()`
- Tests: existing PV tests adapted to power-based assertions

**Step 4H: Remove `currentControllerAdapter`**
- Delete `core/loadpoint_controller.go`
- All current-controlled chargers now use `CurrentController`
- Final cleanup of loadpoint — remove any remaining current-specific code

#### 4.11 The Vehicle Problem — Solved by Host Interface

The loadpoint's `effectiveMinCurrent()` and `effectiveMaxCurrent()` consult:
1. Loadpoint config (`minCurrent`, `maxCurrent`) — owned by controller
2. Vehicle capabilities (`v.OnIdentified().GetMinCurrent()`, `.GetMaxCurrent()`) — queried via `host.GetVehicle()`
3. Charger capabilities (`charger.(api.CurrentLimiter).GetMinMaxCurrent()`) — owned by controller

The `Host` interface (defined in 4.3) solves the vehicle problem cleanly:
- The controller calls `c.host.GetVehicle()` and queries vehicle limits/phases/features directly
- No stale state, no synchronization, no `UpdateVehicle()` calls to orchestrate
- Vehicle connects/disconnects are invisible to the controller; `GetVehicle()` returns nil when no vehicle
- Wake-up on `ErrAsleep` calls `c.host.WakeUpVehicle()` — the loadpoint handles Resurrector lookup
- Charging state calls `c.host.Charging()` — the loadpoint owns charger status

#### 4.12 The `Controller` Interface Extension

The `Controller` interface needs additional read-only accessors so the loadpoint can publish state and delegate API calls:

```go
type Controller interface {
    // ... existing methods ...
    SetOfferedPower(power float64) error
    SetMaxPower() error
    MinPower() float64
    MaxPower() float64
    EffectiveChargePower() float64

    // State accessors for UI/API
    ActivePhases() int
}
```

Note: Phase-specific accessors (`GetPhases`, `SetPhasesConfigured`, etc.) are `CurrentController`-specific. The loadpoint uses type assertions to access them:

```go
if cc, ok := lp.controller.(*chargercontroller.CurrentController); ok {
    lp.publish(keys.PhasesConfigured, cc.GetPhasesConfigured())
    lp.publish(keys.ChargerPhases1p3p, cc.HasPhaseSwitching())
}
```

For `PowerController`, these are not applicable — power devices don't have phases.

### Phase 5: Replace `pvMaxCurrent` with `pvMaxPower` ✅ Covered in Step 4G

### Phase 6: Clean up

- Remove `loadpoint.Controller` interface from `core/loadpoint/api.go`
- Remove `LoadpointControl` methods from chargers (heatpump, mypv, ego, sgready)
- Remove `lp loadpoint.API` fields from chargers
- Move `powerToCurrent`/`currentToPower` from `core/helper.go` to `chargercontroller/` (private)
- Regenerate mocks: `go generate ./api/... ./core/...`
- Update WebSocket key publishing for new state locations
- Update frontend if needed

## State Publishing

The controller publishes state for the UI via a `publish func(key string, val any)` callback injected at creation. This keeps the same WebSocket keys the frontend expects.

For `CurrentController`:
- `keys.ChargeCurrent` — offered current
- `keys.PhasesActive` — active phases
- Phase timer state

For `PowerController`:
- Minimal publishing needed (offered power is the loadpoint's concern)

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
| Vehicle state consistency | Controller queries `host.GetVehicle()` on every call; always reads latest values |
| Power→current→power round-trip in `currentControllerAdapter` | Eliminated once `CurrentController` is complete |
| `optimizePhases` differs from `pvScalePhases` | Keep identical timer durations/thresholds; test equivalence |

## Summary of Complexity Reduction

| Before | After |
|--------|-------|
| Loadpoint manages current, phases, and power | Loadpoint manages power only |
| `pvMaxCurrent()` — 120 lines with phase switching | `pvMaxPower()` — ~50 lines, no phase awareness |
| `setLimit()` — 90 lines with circuit+enable+wake-up | `CurrentController.SetOfferedPower()` — same logic, encapsulated |
| `IntegratedDevice` special cases scattered | `PowerController` struct handles power devices cleanly |
| Chargers need `loadpoint.Controller` to query phases | Chargers implement `api.PowerController`, no loadpoint coupling |
| Power→current→power round-trip | Direct power path, no conversion loss |
| Phase switching embedded in strategy (`pvMaxCurrent`) | Phase switching encapsulated in controller (`optimizePhases`) |
| `effectiveCurrent()` Zoe hysteresis leaked into strategy | `EffectiveChargePower()` hides it behind controller interface |
| ~400 lines of current logic in loadpoint | ~0 lines of current logic in loadpoint |
