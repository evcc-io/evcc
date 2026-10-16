# Absence scheduling: design decisions

## Problem

The optimizer plans charging across multiple days but doesn't know when the vehicle won't be at the charger. Users should be able to declare expected absence windows. This overlaps with, but is not the same as, existing charging plans (hard SoC goals).

## Decisions

- Goals and absences are separate concepts. A goal is a hard commitment; an absence is a soft expectation. A goal does not imply the vehicle leaves afterwards; an absence needs no SoC. They may coincide, but are never linked in data.
- Both are managed per vehicle, in one place, behind one entry point. Users shouldn't need to know our taxonomy to find the right door.
- Absences never block anything. No hard errors, no invalid states. A goal inside an absence produces at most a hint; the planner charges before departure and the entry stays valid. Overlapping absences merge silently.
- Recurring and one-time absences are one entry type with a repeat property. Multi-day trips are just one-time entries ending on a later day. Expired one-time entries disappear on their own; recurring ones live until deleted.
- Absences are labeled by picking from preset occasions, not free text. Scannable, translatable, icon-representable.
- The computed plan result is the primary content; the schedule configuration is secondary (write-once, read-rarely). A weekly calendar visualization supports comprehension but is read-mostly. It navigates to entries, it is not an editing surface. No full calendar widget.
- The plan result is derived from all active entries together; it is never attached to a single entry.
- Entries can be switched off without deleting (e.g. disable "Work" during a vacation week).
- Editing happens inline, no view stacking or native-style navigation. Destructive actions are tucked into the edit context, not exposed on every row.

## Constraints & assumptions

- Mobile first; one responsive component everywhere.
- 15-min time resolution.
- Per-vehicle data, like plans; not per-loadpoint.
- No automation semantics: absences only inform the optimizer's expectations, they trigger nothing.
- Follows existing evcc UI conventions and components (toggles, tabs, icon and type standards, existing plan-chart language).

## Open questions

- Localization of the concept name.
- Behavior when the vehicle is actually connected during a declared absence. Presumably reality wins; confirm optimizer behavior.
- Where absences live in API/config; interaction with vehicle min-SoC settings.
- Should the optimizer eventually learn absence patterns, with manual entries as overrides?
- Dark mode pass.
