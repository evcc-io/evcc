---
description: 'Tidy templates'
tools: ["read", "edit", "search","shell"]
---

## Cleanse parameter defaults

Validate templates files inside `templates/definition/**` against default parameters in `util/templates/defaults.yaml`.
Make sure that templates don't repeat parts of the default parameter definition.

## Reorder parameters after presets

Validate templates files inside `templates/definition/**`. If the template contains parameters and presets, ensure that any parameter contained in a `preset` appears in the template after the `preset`. Reorder such parameters after the presets.
