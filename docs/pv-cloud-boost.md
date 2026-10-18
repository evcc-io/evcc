# PV cloud support for fixed-phase vehicle charging

## Policy

With a journal-capable PV battery, this path applies to fixed-phase loadpoints
(for example, three phases) even when the separate battery boost switch is off. It uses the
existing site `bufferSoc` setting (for example, 50%), not the separate
loadpoint `batteryBoostLimit`. The site `bufferStartSoc` cannot authorize a start
on this path. Without a journal-capable battery, explicit boost retains its
existing activation and SoC-limit settings. Automatic phase-switching retains its existing
implementation. The EV minimum is derived from effective minimum current and
active phases, not hard-coded to 4200 W. At 6 A, 230 V and three phases it is
4140 W. The minimum follows the configured charger and vehicle limits.

- Start and restart require enough PV alone for the minimum EV charging power.
  Existing enable delay remains. Battery support cannot pass the start gate.
- While actually charging, use PV plus the permitted battery discharge budget.
- Preserve the signed PV balance after house demand, other EV consumption and
  reserve. Negative PV balance first consumes battery capacity for the house.
- Pause in the current control cycle when the combined budget is below the EV
  minimum. The normal disable delay, buffer-start setting and minimum-current
  overrides must not keep charging from a planned grid contribution.
- At or below `bufferSoc`, no battery support is budgeted. Continue on PV alone
  if sufficient; otherwise pause. A paused vehicle must pass the PV-only start
  gate again even if the battery recharges above the buffer. A value of zero or
  100 disables automatic buffer support. Legacy explicit boost retains its hold semantics.
- Respect known discharge limits, maximum EV current and downstream circuit
  limits. Missing/non-finite discharge limits mean no planned battery support.
- An unavailable current SoC reading contributes no discharge budget, even when
  the dashboard retains the last measured SoC.
- This changes the PV branch of smart charging. Active optimizer suggestions retain the upstream control path. Explicit fast mode, scheduled
  charging and cheap-tariff overrides keep their existing priorities. It does not
  promise zero instantaneous import during physical response delays.

## Accounting

The measured site balance adds battery discharge back to grid power. Subtracting
that balance from the current measured EV power reconstructs the PV contribution
available to this EV, without counting existing battery discharge as solar.
For boost, auxiliary demand and other EV power remain in the balance rather than
being treated as reclaimable power.

Conceptually (excluding DC adjustments):

```
PV balance = PV production - house - other EVs - reserve
running EV budget = PV balance + allowed battery discharge
```

Do not add measured battery discharge again. Do not clamp a negative PV balance
to zero before adding battery support: that would allocate the house's portion
to the EV too.

The reference weather sequence, without reserve/losses and with 3300 W allowed
discharge, is 4500+3300=7800, 3000+3300=6300, 1500+3300=4800 W, then pause at
500+3300=3800 W. These are power budgets; actual commands follow supported current
steps, so a charger restricted to whole amps may draw less.

## Separate home-battery start setting

The home-battery PV charge start value remains configurable in the GUI and
persisted in evcc settings. 500 W is only its default. The car's minimum power
and the battery's maximum charge/discharge power do not replace this setting.
Tests cover 250, 500, 750 and 1500 W start settings and changing the setting while
battery charging is already active.

## Tests and remaining integration

Simulated tests cover the weather
sequence, start/restart gate, enable delay, exact SoC boundary, PV-only continuation,
house priority, independence from actual battery discharge, reserve, missing
limits/SoC, immediate pause and EV current cap.

This budget controller does not send new manual sonnen discharge commands. It
relies on the battery supplying demand through its native regulation and on an
accurate allowed discharge limit. Simulated tests do not verify the physical
battery response. The action journal and GUI confirmation are implemented separately; see
[PV battery control](active-pv-battery-charging.md). Manual or unobserved batteries
contribute no discharge capacity and their charging demand is not reclaimable PV.
The coordinated path uses the allocation and handoff rules below. Legacy explicit
boost still returns owned PV charging to automatic mode. No token or production
database was accessed during development.

## Shared allocation and handoff

The 12 October logs showed a repeated allocation error: at 13:32:32 the vehicle
offer increased from 6 to 8.11 A (about 1456 W), while the same measured export
caused a 1451 W home-battery charge command. The new accounting reserves positive
offer-minus-measurement differences across connected, enabled vehicles before
allocating battery charging. A reduction creates no free budget until measured.
Other vehicle offers and unmeasured journal commands are likewise reserved when
computing a vehicle's budget. Outstanding release commands remain reserved until
their restoration is confirmed. This is conservative: a vehicle that draws less
than its offer can leave some export unallocated. No timeout guesses that an
accepted offer has ceased to be actionable.

Manual charging is never counted as immediately reclaimable PV. When export plus
charging owned by our journal would cover a waiting vehicle's minimum (after the
reserve and respecting battery priority), the battery returns to automatic mode
first. Allocation then waits for measurements and journal confirmation. Foreign
manual charging is not released. A running vehicle with current headroom gets
priority over new battery charge requests; at the EV current cap, remaining
export may charge the battery above its configured 500 W default start value.

Tests reproduce the 1551 W export / 8.11 A offer case, delayed meter response in
both directions, multiple EV reservations, the journal handoff, no acquisition
of foreign control, priority SoC, current caps, 50% boundary with boost disabled,
and high-SoC PV-only start/restart. These tests cannot establish hardware timing.
