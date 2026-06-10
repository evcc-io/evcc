# Power-first loadpoint control — refactoring plan

## Goal

Loadpoint currently controls charging in the current (ampere) domain while site works on
available power. This refactoring makes loadpoint work on **power exclusively** and delegates
all current handling to a separate `CurrentController`.

## End-state contract

Loadpoint computes a single target power per update cycle and talks to the controller through
a minimal contract:

- `SetPower(power float64) error` — the only control verb
- `MinPower()` / `MaxPower()` — capability envelope queries

The controller owns *all* current math: phase scaling, current rounding, vehicle current
limits, and charger writes. `api.PowerController` is renamed accordingly: its setter becomes
`SetPower(power float64) error` (instead of the `MaxCurrent`-style `MaxPower(power)`), freeing
the `MinPower`/`MaxPower` names for the envelope.

### Envelope semantics

Capability envelope, same behaviour as today:

- when automatic phase switching is allowed, the envelope spans 1p-min … 3p-max
- when phase configuration is locked (fixed 1p/3p), the envelope reflects the locked
  configuration

### Setpoint semantics

- `SetPower(0)` means *disable charging*; the controller executes the enable/disable and
  wake-up mechanics
- a positive setpoint below the feasible minimum is clamped **up** to `MinPower()` — a
  positive value expresses "charging shall happen", and an envelope race must never silently
  stop charging; the failure mode is a few hundred extra watts for one cycle

### Policy/execution split

Loadpoint keeps (power-domain policy):

- PV enable/disable hysteresis (`pvTimer`, watt-denominated thresholds); loadpoint only ever
  emits `0` or a value ≥ `MinPower()`
- welcome-charge and climater minimum floors, expressed as `max(target, MinPower())` — the
  controller's `current == 0` sentinel coda is removed; the controller never knows *why* it
  is asked for minimum power

Controller owns (current-domain execution):

- the 1p↔3p phase decision including `phaseTimer` and scale hysteresis; phase state remains
  published, but as controller telemetry
- `enabled`, `offeredCurrent`, `phases` as private state (after the structural inversion,
  see below)

### Native power chargers

Chargers natively implementing `api.PowerController` (DC, OCPP watt profiles) are a
**follow-up**. This series only guarantees the seam is shaped so a native implementation can
slot in by supplying its own envelope.

## Steps

1. **This PR** — rework in place (new commits, no force-push):
   - collapse the transitional intent vocabulary (`chargeMinimum`, `fastCharging`,
     `feedInCharging`, `pvCharging`, `enforcePhases`) into `SetPower` + envelope
   - move welcome/climate floors into loadpoint power policy
   - relocate the phase decision and `phaseTimer` into the controller
   - the controller keeps embedding `*Loadpoint` for plumbing for now
2. **Follow-up PR** — structural inversion: controller-owned state (`enabled`,
   `offeredCurrent`, `phases`, `phaseTimer`), explicit narrow dependencies (charger, clock,
   logger, publish hook, vehicle-limit provider), no `*Loadpoint` embedding.
3. **Follow-up** — activate native `api.PowerController` chargers.

## Behaviour changes

Structural steps are strictly behaviour-preserving. Deliberate fixes are permitted in three
known wart areas, each as its **own flagged commit/PR with a test pinning the new behaviour**;
any unflagged behaviour diff is by definition a regression:

- floor/hysteresis interplay (welcome/climate floor vs. PV enable/disable timers)
- phase-scaling edge cases (threshold flapping, timer resets on mode change)
- min-current rounding/clamping (incl. the feed-in MinPV configured-vs-effective-min
  asymmetry)

## Verification

- the existing loadpoint test suite passes unmodified, except where a flagged wart commit
  deliberately changes pinned behaviour
- new controller-isolation unit tests via the existing test helper: envelope under
  locked/auto phases, sub-minimum clamping, `SetPower(0)` disable, phase-timer hysteresis

## Non-goals / unchanged

- user-facing config and API stay current-denominated (`minCurrent`, `maxCurrent`, `phases`)
  and route to the controller
- vehicle-derived current limits become controller envelope inputs
- per-cycle charger write deduplication is preserved
