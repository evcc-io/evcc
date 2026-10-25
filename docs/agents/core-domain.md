# Core Domain: Site, Loadpoint, and the Control Loop

## Object Hierarchy

```
Site (orchestrator — core/site.go)
├── Meters: Grid, PV[], Battery[], Auxiliary[], External[]
├── Tariffs: Grid, FeedIn, CO2, Solar
├── Coordinator (vehicle <-> loadpoint assignment)
├── Prioritizer (power allocation fairness)
└── Loadpoints[] (core/loadpoint.go)
    ├── Charger      (api.Charger — hardware controller)
    ├── Vehicle      (api.Vehicle — EV battery state via cloud API)
    ├── ChargeMeter  (api.Meter — AC power at charger)
    └── Circuit      (optional — electrical domain limits)
```

## Key Interfaces (api/api.go)

### Meter
- `Meter` — `CurrentPower() (float64, error)` — watts
- `MeterEnergy` — `TotalEnergy() (float64, error)` — kWh
- `PhaseCurrents` / `PhaseVoltages` / `PhasePowers` — per-phase readings

### Battery
- `Battery` — `Soc() (float64, error)` — 0-100%
- `BatteryCapacity` — kWh
- `BatteryController` — set charge/discharge/hold mode

### Charger
- `Charger` — `Status()`, `Enabled()`, `Enable(bool)`, `MaxCurrent(int64)`
- `ChargerEx` — milliamp-precision current via `MaxCurrentMillis(float64)`
- `PhaseSwitcher` — `Phases1p3p(int) error`
- `ChargeRater` — `ChargedEnergy() (float64, error)`
- `ChargeTimer` — `ChargeDuration() (time.Duration, error)`

### Vehicle
- `Vehicle` — `Soc()`, `Capacity()`, `Identifiers()`, `Phases()`, `OnIdentified()`
- `VehicleRange`, `VehicleOdometer`, `VehicleClimater`, `VehiclePosition`
- `ChargeController` — remote start/stop on vehicle
- `CurrentLimiter` — `GetMinMaxCurrent()` for vehicle-side current limits
- `CurrentController` — some vehicles (Tesla, Fiat) also implement `MaxCurrent()` to set charge current from the vehicle side

## Charge Modes

| Mode | Behavior |
|------|----------|
| `OFF` | Disabled (unless welcome charge) |
| `SMART` | Ramp current with solar surplus; fast if cheap tariff or plan requires |
| `NOW` | Max current immediately |

Always charge (`alwaysCharge`: off/on/once) is a per-loadpoint option for smart mode: never pause, keep at least min current. `once` resets on disconnect. `pv` and `minpv` are deprecated write-only aliases for smart without/with always charge.

## Charge States (IEC 61851)

- `A` — not connected
- `B` — connected, not charging
- `C` — connected, charging

## The Control Loop (Site.update — runs every N seconds)

```
1. Update all meters (grid, PV, battery, aux)
2. For each loadpoint: UpdateChargePowerAndCurrents()
3. Calculate site power balance:
   sitePower = gridPower + batteryPower + excessDCPower
             + residualPower - auxPower - flexiblePower
4. Apply battery priority rules (prioritySoc, bufferSoc)
5. Get tariff rates
6. For EACH loadpoint: Update(sitePower, ...)
   ├── Read charger status
   ├── Detect/identify vehicle
   ├── Check plan requirements (minSOC, target time)
   ├── Check limits (limitSOC, limitEnergy)
   ├── MODE switch -> calculate target current
   ├── Cap at maxCurrent, respect circuit limits
   ├── Send MaxCurrent() to charger
   └── Record metrics
7. Push updates to WebSocket + metrics
```

The loop is stateless per cycle: always re-reads actual state, calculates
optimal current, sends single command. Resilient to restarts and missed updates.

## PV Surplus Charging (pvMaxCurrent in core/loadpoint.go)

```
1. Read effective min/max current limits
2. Reduce sitePower by battery boost power
3. Consider phase switching (1p <-> 3p) if supported
4. deltaCurrent = powerToCurrent(-sitePower, activePhases)
   targetCurrent = effectiveCurrent + deltaCurrent
5. Below minCurrent -> start disable timer (default 3 min)
6. Surplus returns -> start enable timer (default 1 min)
7. Cap at maxCurrent
```

## Battery Priority Rules

| Setting | Effect |
|---------|--------|
| `prioritySoc` | Below this: battery charges first, EV gets 0 |
| `bufferSoc` | Above this: EV can draw from battery reserves |
| `bufferStartSoc` | Above this: EV charging can begin even if importing |

None of these apply to a loadpoint the optimizer controls: the optimizer plans the home battery, so a stop gives the battery priority and a charge takes the surplus (`loadpointSitePower`, `#34606`). A charge the home battery covers is pinned to its setpoint instead, the pv loop would count the discharge as deficit (`surplusRegime`, `#34809`).

## Effective Price Calculation

```
greenShare = (max(pvPower,0) + max(batteryPower,0)) / totalChargePower
effectivePrice = gridPrice * (1 - greenShare) + feedInPrice * greenShare
```

## Concurrency Model

- **Site** owns `RWMutex` for its state (meters, battery, tariffs)
- **Loadpoint** owns `RWMutex` for its state (charger, vehicle, current)
- **Coordinator** owns `RWMutex` for vehicle <-> loadpoint tracking
- No global locks — ordering prevents deadlocks

### Channels

| Channel | Scope | Buffer | Purpose |
|---------|-------|--------|---------|
| `valueChan` | Site | Unbounded (`chanx.NewUnboundedChan`) | State changes -> DB + UI (ordering) |
| `lpUpdateChan` | Site | 1 | Early loadpoint update requests |
| `pushChan` | Loadpoint | 16 | User notifications, queued via `valueChan` so the message renders the state at event time |

## Tariff Integration

Types: `TariffUsageGrid`, `TariffUsageFeedIn`, `TariffUsageCo2`, `TariffUsagePlanner`, `TariffUsageSolar`

### Smart Features
- **Cheap-tariff override** — rate below threshold -> fast charge
- **Smart feed-in** — feed-in rate above threshold -> prioritize export
- **Planner** (`core/planner/planner.go`) — finds cheapest time slots for target SOC/energy by deadline
  - `optimalPlan()` — cheapest non-contiguous slots
  - `continuousPlan()` — cheapest continuous window (fallback)

## Optimizer Suggestions (core/site_optimizer.go)

The optimizer (`optimizer.evcc.io`, sponsor-gated) solves a linear program over tariff slots. Every solved "battery" (home battery, vehicle/loadpoint) gets a `types.Suggestion` for the slot covering now. In `Automatic` mode these suggestions drive control; otherwise they are advisory (UI + notifications).

### Request side: what becomes a battery

- **Home battery meter**: `batteryRequest`. `controllable` = `api.HasCap[api.BatteryController]`. Only controllable batteries get `charge_from_grid` / `discharge_to_grid`.
- **Loadpoint with vehicle**: `loadpointRequest`, `detail.Type = vehicle`, `detail.loadpoint = &id`, always `controllable`. Needs vehicle capacity (or a session energy limit) to model as storage. `ModeOff` sets `CMax = 0`; `ModeNow` and plans produce a `PDemand` / `SGoal`.
- **Unmodellable loadpoints** (no vehicle, no capacity and no energy limit, or heaters) are added to the home load (`addUnmodelledLoad`) and get no suggestion.
- Disconnected loadpoints are skipped entirely.

Device key (`batteryDetail.key()`): `battery:<name>` or `loadpoint:<id>`; empty key = cannot act.

### Result side: `applyOptimizerResult`

Runs after each solve, and again every cycle via `reapplySuggestions` when the active slot changes (re-derived from the cached solve, no new request). Expiry is only evaluated on that slot change: if the horizon is passed or no solve completed for `2 * tariff.SlotDuration`, `clearSuggestions()` drops suggestions, battery forecast and notification state. Within an unchanged slot the cached advice stays active regardless of age. A failed solve keeps the previous advice so controlled devices are not released for one cycle.

1. `slot = schedule.activeSlot(now)`, `slotHours` = its duration. Grid import/export and all home-battery discharge for that slot are summed.
2. `currentSlotSuggestion(detail, req, res, slot, gridImport, gridExport, slotHours)` per battery. Because the LP is linear, the slot result sits at an operating-range extreme and maps cleanly onto discrete modes. All thresholds use `suggestionThreshold = 50 W` to ignore solver noise.

**Home battery** (`Action` is an `api.BatteryMode` string):

| Condition (slot values, W)                         | Action       | Meaning                             |
| -------------------------------------------------- | ------------ | ----------------------------------- |
| charge > thr and grid importing                    | `charge`     | grid charging                       |
| idle and grid importing and soc > `SMin`           | `hold`       | discharge deliberately withheld     |
| idle and grid exporting and soc < `SMax`           | `holdcharge` | surplus exported instead of charged |
| discharge > thr and grid exporting                 | `discharge`  | battery-to-grid                     |
| otherwise (incl. idle at a soc bound, PV charging) | `normal`     | inverter self-consumption logic     |

`idle` = charge and discharge both <= thr. At a soc bound nothing is withheld (the bound does it), hence `normal`.

**Loadpoint/vehicle**: `charge` if charge > thr, else `stop`. `Suggestion.Battery` is set to the planned home-battery discharge so the loadpoint can tell surplus from battery-fed charging.

3. Suggestions are stored only for controllable devices with a key (`site.suggestions`). Loadpoints with a plan goal (`SGoal`) also receive `optimizerPlan` (the remaining charging slots priced at `PN`) via `setOptimizerPlan`, used by `OptimizerPlan()` for the planner UI.
4. `publishBattery` / `publishSuggestions` publish to UI and hand loadpoint suggestions to `lp.setSuggestion`. `Actionable` is evaluated on every read (`site.suggestion`) as `Action != currentAction`, where the current action is `site.batteryAction()` for batteries (the battery mode, `unknown` reads as `normal`) and `loadpointCurrentAction` (`charge` when enabled and goal not reached, else `stop`).
5. `diffSuggestions` emits one `suggestion` messenger event per device whose actionable action changed since the last run; non-actionable or vanished devices are pruned so a later change re-notifies.

### Control side: where suggestions take effect

Gated by `OptimizerAutomatic()`: `off` (advisory only), battery-only, or `full` (also loadpoints). `site.Automatic()` additionally requires optimizer enabled + sponsor.

**Home battery** (`requiredBatteryMode`, `core/site_battery.go`), in priority order:

- external mode (API) wins.
- `Automatic()` and `unmodelledCharging(rate)`: a loadpoint the optimizer does not control is fast charging or on the smart cost limit. The suggestion did not account for that load, so force `hold`.
- `Automatic()`: `batterySuggestionMode()` takes the first controllable battery's suggestion (`api.BatteryModeString(s.Action)`); an unknown action releases to `normal`; no suggestion releases to `normal` only if evcc had modified the mode (`batteryModeModified`), otherwise the mode is left untouched. This replaces the grid charge limit and discharge control. TODO: per-battery modes.
- not automatic: legacy grid charge / discharge control / feed-in discharge rules.

**Loadpoint** (`activeSuggestion`, `core/loadpoint_optimizer.go`): non-nil only if `optimizerControlled()` (full automatic, not a heater or integrated device, vehicle with capacity > 0) and the suggestion is younger than `suggestionMaxAge = 2 * SlotDuration` (a stalled optimizer must not keep the loadpoint gated). With an active suggestion:

- `ModeSmart` strategy switch calls `optimizerCharging(s)` and skips the smart cost / feed-in price limits.
  - `surplusRegime` (charge below max power, no grid flow, no battery discharge): not handled, the PV loop tracks measured surplus instead of the forecast.
  - `stop` without always-charge: not handled, PV loop may still enable on measured surplus.
  - `charge` at full power or above what the active phases deliver: `fastCharging()`.
  - `charge` at a limited setpoint: `setLimit(PowerToCurrent(s.Charge))`, or `minCharging()` below min current.
  - `stop` with always-charge: `minCharging()`.
- Planner is reduced to a backstop: `plannerActive = planDeadlineCritical()`.
- Session `LimitEnergy` / `LimitSoc` do not stop charging while the optimizer is driving a plan goal above them.
- `loadpointSitePower`: priority soc adjustment is applied, and the battery's discharge is subtracted from the surplus unless the action is `charge` (the optimizer plans the home battery, so `bufferSoc` / `bufferStartSoc` are also disabled).

Every cycle ends with `site.publishSuggestions()` so `Actionable` reflects the updated loadpoint state.

## Key File Locations

- `api/api.go` — all core interfaces
- `core/site.go` — Site orchestrator + control loop
- `core/loadpoint.go` — Loadpoint state machine (pvMaxCurrent, mode switch)
- `core/site_battery.go` — battery priority logic, battery mode from suggestion
- `core/site_optimizer.go` — optimizer request, suggestion mapping, reapply/expiry
- `core/loadpoint_optimizer.go` — loadpoint suggestion gate and charging strategy
- `core/site_tariffs.go` — tariff integration
- `core/planner/planner.go` — charge time optimization
- `core/prioritizer/prioritizer.go` — power allocation across loadpoints
- `core/circuit/circuit.go` — electrical domain limits
