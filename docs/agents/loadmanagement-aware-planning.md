# Loadmanagement-aware planning (design + TODOs)

Status: **draft / tracking**. Design for making tariff planning respect shared
circuit (load management) capacity across multiple loadpoints. No behaviour
change yet — this document scaffolds the work.

## Problem

Planning is per-loadpoint and unaware of *shared* circuit capacity. Each
loadpoint's `plannerActive()` → `GetPlan()` → `planner.Plan(requiredDuration, …)`
(`core/loadpoint_plan.go`, `core/planner/planner.go`) independently picks the
globally cheapest slots to meet its own target.

Load management does reach planning in one narrow way: `requiredDuration`
derives from `EffectiveMaxPower()`, which clamps to
`min(loadpoint max, circuit.GetMaxPower())` (`core/loadpoint_effective.go`). So a
single loadpoint won't plan beyond the circuit's **total** max. That total-clamp
is acceptable as-is — the gap is **shared, per-slot** capacity:

- each loadpoint assumes it owns the whole circuit — no subtraction of other
  loadpoints' concurrent draw or plans;
- only the loadpoint's **direct** circuit is considered, not the parent chain;
- per-phase **current** limits and **dynamic** limits (`getMaxCurrent`) are
  ignored;
- it is a single scalar clamp on total duration, not a per-time-slot allocation,
  so the planner never sees how much budget is free in a given slot; minSoc
  forced charging is invisible to it.

Circuit enforcement is reactive, lagging feedback. Loadpoints are controlled
sequentially, one per tick (`Run` → `loopLoadpoints` → single `site.update(lp)`),
each after a fresh `circuit.Update`. `setLimit` clamps a loadpoint's current to
the remaining headroom via `circuit.ValidateCurrent` / `ValidatePower`
(`core/circuit/circuit.go`). Because it only reacts to already-measured draw, a
loadpoint that ramps into a slot the planner over-committed is throttled a cycle
late — it can never un-miss the resulting deadline.

### Failure chain

1. LP-A and LP-B each independently pick the same cheap slot X (both saw it as
   cheapest). Combined demand `2 × Pmax > circuit Pmax`, yet each plan looks
   individually feasible against the circuit's total.
2. In slot X both ramp up; the reactive clamp throttles them a cycle late,
   so combined draw briefly exceeds the limit before it catches up.
3. Throttled below plan, they under-deliver in their cheap slots → reach
   deadline short of target. Nothing reconciled the plans up front.
4. minSoc compounds it: `minSocNotReached()` (`core/loadpoint.go`) forces fast
   charging now (`IsFastChargingActive`), consuming budget another loadpoint's
   plan assumed it had — invisible to that plan. Simultaneous ramp of planned +
   forced → transient `over current detected` warning
   (`core/circuit/circuit.go`), and with a slow real meter, a breaker trip.

Root gap: plans optimise against the tariff and clamp only to the circuit's
**total** capacity; the binding constraint is **shared** circuit capacity
**per slot over time**, which no planner sees.

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
- [x] Warn (in the planner) when a plan actually misses its goal — the required
      duration no longer fits before the target, so charging will overrun.
      Warned once per target. This surfaces load-management-induced misses too:
      a throttled loadpoint charges slower than `maxPower` assumed, so its plan
      eventually overruns.

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
- [ ] Allocate the shared budget in whole `EffectiveMinPower()` chunks
      (semi-continuous, same rule as `evcc-io/optimizer#91`): a charger runs at
      `>= effectiveMinPower` or off, never below. So at most
      `floor(circuitBudget / effectiveMinPower)` planned sessions can run in a
      slot; a loadpoint that cannot be granted its `effectiveMinPower` must be
      left off for that slot and its plan shifted, not run sub-minimum.
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
- Semi-continuous allocation makes some parallel-plan sets simply infeasible:
  when `floor(circuitBudget / effectiveMinPower)` is fewer than the loadpoints
  that must charge at once, a plan cannot fit and the goal-miss warning fires.
  Allocation cannot hand out sub-`effectiveMinPower` slivers to paper over this.
