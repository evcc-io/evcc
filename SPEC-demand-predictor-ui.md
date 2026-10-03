# Spec: Demand predictor selector for heating loadpoints

> **TRANSIENT — remove before merging.**
> Tracks implementation of the UI discussed in PR #34342.

## Goal

Allow users to select the demand forecast strategy for a heating loadpoint
at runtime via the loadpoint settings modal, without editing `evcc.yaml`.

## Background

PR #28232 introduced three demand forecast strategies selectable via
`predictor:` in `evcc.yaml`:

| YAML value | Behaviour |
|---|---|
| *(unset)* | 28-day per-slot average (default) |
| `DemandWeekday` (or `demandweekday`) | Same-weekday average over 4 weeks |
| `DemandTemperature` (or `demandtemperature`) | 7-day average scaled by outdoor temperature forecast |

Both casings are accepted by the enumerator. The canonical Go names are
`DemandWeekday` and `DemandTemperature`.

The YAML key sets a static default baked into the charger template.
This spec adds a runtime override so users can switch strategies from the UI.

## User-facing behaviour

A new `<select>` row labelled **Demand forecast** appears in the Heating
section of the loadpoint settings modal, directly below "Min. temperature".
It is visible only for heating devices (`chargerFeatureHeating = true`).

### Options

| Display name | Best used for |
|---|---|
| **Average** | Devices with consistent usage throughout the day. |
| **Weekday** | Devices that follow a weekly household rhythm, like warm water or pool heaters. |
| **Temperature** | Room heating that follows outdoor temperature, like heat pumps or floor heating. |

### Hint text

Below the dropdown, a single line of muted text explains the selected option
(same text as the "Best used for" column above, adapted as a sentence).

### Effective value

The dropdown always shows the **effective** predictor:
- If a runtime override is set: show that.
- Otherwise: infer from the charger feature flags
  (`DemandTemperature` → Temperature, `DemandWeekday` → Weekday, neither → Average).

### Temperature option availability

If no temperature tariff (`TariffUsageTemperature`) is configured, the
**Temperature** option is still shown but a warning hint is displayed:
*"Requires a temperature tariff to be configured."*
The option remains selectable (the backend already degrades gracefully to
the uncorrected 7-day average).

---

## Implementation

### 1. `core/loadpoint/config.go`

Add `DemandPredictor string` to `DynamicConfig` with json tag `"demandPredictor"`.

Add `lp.SetDemandPredictor(payload.DemandPredictor)` to `Apply()`.

### 2. `core/keys/loadpoint.go`

```go
DemandPredictor = "demandPredictor"  // active demand predictor for heating loadpoints
```

### 3. `core/loadpoint.go`

Add field to the `Loadpoint` struct (alongside `solarShare`):

```go
demandPredictor string  // runtime override: "daily", "weekday", "temperature", or ""
```

In the `publishSettings()` / initial publish block (where `keys.SolarShare` is published):

```go
lp.publish(keys.DemandPredictor, lp.demandPredictor)
```

In `NewLoadpointFromConfig` defaults: no default needed — empty string means
"use charger feature flags" (same as existing behaviour).

### 4. `core/loadpoint/api.go`

Add to the `API` interface (near `GetSolarShare` / `SetSolarShare`):

```go
// GetDemandPredictor returns the runtime demand predictor override.
// Empty string means use the charger template default.
GetDemandPredictor() string
// SetDemandPredictor sets the demand predictor override ("daily", "weekday", "temperature", or "").
SetDemandPredictor(predictor string)
```

Regenerate mock: `go generate ./core/loadpoint/`.

### 5. `core/loadpoint_load_predictor.go`

Add getter and setter on `*Loadpoint`:

```go
func (lp *Loadpoint) GetDemandPredictor() string {
    lp.RLock()
    defer lp.RUnlock()
    return lp.demandPredictor
}

func (lp *Loadpoint) SetDemandPredictor(predictor string) {
    lp.Lock()
    lp.demandPredictor = predictor
    lp.Unlock()
    // publish outside the lock: publish enqueues on a channel and must not
    // be called while holding the loadpoint mutex (same pattern as SetSolarShare).
    lp.publish(keys.DemandPredictor, predictor)
}
```

Add unexported helper:

```go
// effectiveDemandPredictor returns the active predictor, consulting the
// runtime override first and falling back to the charger feature flags.
// Uses GetDemandPredictor() to read the field safely without an additional lock.
func (lp *Loadpoint) effectiveDemandPredictor() string {
    if p := lp.GetDemandPredictor(); p != "" {
        return p
    }
    switch {
    case lp.chargerHasFeature(api.DemandTemperature):
        return "temperature"
    case lp.chargerHasFeature(api.DemandWeekday):
        return "weekday"
    default:
        return "daily"
    }
}
```

Update `demandProfile()`: replace the two `chargerHasFeature` calls with
`lp.effectiveDemandPredictor()`.

Update `demandProfileWeekday()`: replace the `chargerHasFeature(api.DemandWeekday)`
guard with `lp.effectiveDemandPredictor() == "weekday"`. Also relax the outer
guard from `!lp.chargerHasFeature(api.DemandWeekday)` to `!lp.chargerHasFeature(api.Heating)`
so a non-template weekday override is honoured.

### 6. `server/http.go`

Add to the per-loadpoint route map:

```go
"demandPredictor": {"POST", "/demandpredictor/{value:[a-z]+}", stringHandler(pass(lp.SetDemandPredictor), lp.GetDemandPredictor)},
```

Accepted values: `daily`, `weekday`, `temperature`. Note that `"daily"` is a
new string introduced by this feature — it has no corresponding `api.Feature`
flag. Posting `daily` explicitly selects the 28-day average and clears any
active weekday or temperature override. An empty-string reset via HTTP is not
needed because `daily` is functionally equivalent.

### 7. `assets/js/types/evcc.ts`

Add to `UiLoadpoint`:

```ts
/** Runtime demand predictor override. Empty string means template default. */
demandPredictor: string;
```

### 8. `assets/js/components/Loadpoints/DemandPredictorDropdown.vue` *(new)*

Options API component. Props:

```ts
loadpointId: string
demandPredictor: string                    // from WS state key "demandPredictor"
chargerFeatureDemandTemperature: boolean   // from WS state
chargerFeatureDemandWeekday: boolean       // from WS state
tariffTemperature: Number | undefined      // from WS state; undefined means no temperature tariff configured
```

Computed `effectivePredictor`: mirrors the Go helper — override → feature flags → `"daily"`.

Renders a `<select>` with three `<option>` elements. On `@change`, posts to
`api.post("loadpoints/" + loadpointId + "/demandpredictor/" + value)`.

Hint line below the select shows the description for the currently selected option.
When `effectivePredictor === "temperature" && tariffTemperature === undefined`, show warning hint instead.

### 9. `assets/js/components/Loadpoints/SettingsModal.vue`

Inside the `<div v-if="heating">` block, after the `mintemp` row:

```html
<div class="mb-3 row">
  <label :for="formId('demandpredictor')" class="col-sm-4 col-form-label pt-0 pt-sm-2">
    {{ $t("main.loadpointSettings.demandPredictor.label") }}
  </label>
  <DemandPredictorDropdown
    :id="formId('demandpredictor')"
    :loadpoint-id="id"
    :demand-predictor="loadpoint?.demandPredictor ?? ''"
    :charger-feature-demand-temperature="loadpoint?.chargerFeatureDemandTemperature ?? false"
    :charger-feature-demand-weekday="loadpoint?.chargerFeatureDemandWeekday ?? false"
    :tariff-temperature="tariffTemperature"
    class="col-sm-8 col-lg-4 pe-0"
  />
</div>
```

`tariffTemperature` follows the existing pattern used for `tariffGrid`,
`tariffCo2`, etc. — a `Number` prop on `SettingsModal` that is `undefined`
when no temperature tariff is configured. Add it alongside the existing
`tariffGrid: Number` prop. Pass it through from `Loadpoint.vue`, which
receives it from `Site.vue` via the same tariff prop chain.

### 10. `i18n/en.json` and `i18n/de.json`

Add under `main.loadpointSettings`:

```json
"demandPredictor": {
  "label": "Demand forecast",
  "daily":       { "description": "Best for devices with consistent usage throughout the day." },
  "weekday":     { "description": "Best for devices that follow a weekly household rhythm, like warm water or pool heaters." },
  "temperature": { "description": "Best for room heating that follows outdoor temperature, like heat pumps or floor heating." },
  "noTempTariff": "Requires a temperature tariff to be configured."
}
```

German (`de.json`):

```json
"demandPredictor": {
  "label": "Verbrauchsprognose",
  "daily":       { "description": "Am besten für Geräte mit gleichmäßigem Tagesverbrauch." },
  "weekday":     { "description": "Am besten für Geräte mit wöchentlichem Rhythmus, z. B. Warmwasser oder Pool." },
  "temperature": { "description": "Am besten für Raumheizungen, die der Außentemperatur folgen, z. B. Wärmepumpen oder Fußbodenheizung." },
  "noTempTariff": "Erfordert einen konfigurierten Temperatur-Tarif."
}
```

---

## What is not in scope

- No change to the YAML `predictor:` key or feature flags.
- No new DB column — `DemandPredictor` rides `DynamicConfig` / the existing
  loadpoint settings persistence mechanism.
- No change to `evaluate_predictors.py`.
- The `profilePercentile` / median toggle (PR #33700) is a separate feature.
