# Loadmanagement-aware planning (design + TODOs)

Status: **draft / tracking**. Design for making tariff planning respect shared
circuit (load management) capacity across multiple loadpoints. No behaviour
change yet — this document scaffolds the work.

## Problem

Planning is per-loadpoint and circuit-blind. Each loadpoint's
`plannerActive()` → `GetPlan()` → `planner.Plan(requiredDuration, …)`
(`core/loadpoint_plan.go`, `core/planner/planner.go`) independently picks the
globally cheapest slots to meet its own target by its own deadline. No
loadpoint knows the shared circuit budget or the other loadpoints' plans.

Circuit enforcement is reactive and order-greedy, not priority-aware.
`setLimit` (`core/loadpoint.go`, `Loadpoint.setLimit`) calls
`circuit.ValidateCurrent` / `ValidatePower` (`core/circuit/circuit.go`), which
caps each loadpoint's delta to the remaining headroom
(`potential = maxCurrent - c.current`) where `c.current` is the running sum of
already-processed loadpoints **in site iteration order**. First loadpoint to
run grabs budget; later ones get the remainder. Priority drives PV-surplus
allocation, not this clamp.

### Failure chain

1. LP-A and LP-B each independently pick the same cheap slot X (both saw it as
   cheapest). Combined demand `2 × Pmax > circuit Pmax`.
2. Plans assumed full power in slot X. At runtime the circuit clamps them —
   each gets roughly half.
3. Both under-deliver in their cheap slots → reach deadline short of target.
   The plan was infeasible w.r.t. the circuit and nothing reconciled it.
4. minSoc compounds it: `minSocNotReached()` (`core/loadpoint.go`) forces fast
   charging now (`IsFastChargingActive`), consuming budget another loadpoint's
   planned charging assumed it had. Simultaneous ramp of planned + forced
   before the next control cycle clamps → transient `over current detected`
   warning (`core/circuit/circuit.go`), and with a slow real meter, a breaker
   trip.

Root gap: plans optimise against the tariff; the binding constraint is shared
circuit capacity over time, which no planner sees.

## Options

- **A — Site-level joint scheduler (global optimum).** One site planner
  allocates per-slot circuit capacity across all loadpoints jointly. MILP /
  greedy scheduling; overlaps the external `optimizer` service. Most correct,
  largest change. Deferred.
- **B — Priority-ordered planning with a capacity ledger (recommended core).**
  Reuse the per-loadpoint planner; make slot availability shared via a
  per-circuit, per-slot residual-capacity ledger. Reserve forced load first,
  then plan loadpoints in priority order against the remaining budget.
- **C — Priority-aware runtime clamp only (complement).** Make the circuit
  clamp deterministic (forced > plan-active > priority) instead of iteration
  order, and feed "throttled" back so an LP extends its effective
  `requiredDuration`. Reactive; cannot un-miss a deadline, but valuable under B.

## Decisions (defaults, revise as needed)

- Priority source: reuse the existing loadpoint `priority` field to order
  planning. **[default]**
- minSoc vs cost: forced load (minSoc / `ModeNow`) reserves circuit budget
  before cost-optimal planning. **[default]**
- Scope: power-only in Phase 2; per-phase current + phase switching later.
  **[default]**
- Limits: static circuit limits first; dynamic (`getMaxCurrent` provider) later.
  **[default]**

## Phases / TODOs

### Phase 0 — Reproduce + observe (no behaviour change)
- [ ] Test harness: 2 loadpoints on one circuit (`maxPower < 2 × Pmax`), one
      with minSoc, both plan-charging the same cheap slot; assert the current
      bug (combined clamp → deadline miss / over-current warning).
- [ ] Log/metric when the circuit clamps a **plan-active** or **minSoc**
      loadpoint, to detect this in the wild.

### Phase 1 — Deterministic priority-aware clamp (Option C)
- [ ] Order circuit-budget allocation by (forced > plan-active > priority) in
      the site/circuit update instead of iteration order.
- [ ] Regression test: contention resolves toward the loadpoint that needs it,
      deterministically.

### Phase 2 — Capacity ledger + priority-ordered planning (Option B)
- [ ] Per-circuit, per-slot residual-capacity ledger, mirroring the parent/child
      circuit tree (like `ValidatePower` recursion).
- [ ] Extend `planner.Plan` / `optimalPlan` to accept a per-slot power cap.
      Today a slot equals full Pmax; a capped slot delivers
      `min(Pmax, residual) × slotLen`, so a partially-available slot must spill
      the plan into more slots. This is the core rework.
- [ ] Site orchestration: reserve forced load → plan loadpoints in priority
      order against the ledger, subtracting each reservation.
- [ ] Tests: joint feasibility (Σ planned power/slot ≤ circuit limit), deadline
      met for all loadpoints, minSoc reserved first.

### Phase 3 — Joint optimisation (Option A), only if greedy proves insufficient
- [ ] Extend the `optimizer` service with a shared-capacity constraint; keep B
      as the local fallback.

## Risks / hard parts

- `optimalPlan` equates a slot with full power; per-slot capping is the core
  rework and needs careful coverage.
- Circuit hierarchy (parent/child) and multiple circuits — the ledger must
  mirror the tree.
- Phases: circuit current limit is per-phase; LP power↔current depends on
  active phases, which can switch mid-plan.
- Dynamic circuit limits vary over time; planning needs a forecast that may not
  exist. Static-only first.
- Loadpoints on no circuit stay unconstrained (current behaviour).
