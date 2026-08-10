# Dynamic Vehicle Minimum SoC from Solar Forecast

## Summary

Add an optional policy that dynamically raises a vehicle's minimum state of charge (SoC) according to the expected solar production during the next 72 hours.

The initial forecast states are:

| State | Forecast production during next 72 hours |
| --- | ---: |
| Low | Less than 5 kWh |
| Medium | At least 5 kWh and less than 15 kWh |
| High | At least 15 kWh |

Each vehicle maps these states to configurable minimum SoC values. A user may, for example, require a higher minimum charge when little solar production is expected and accept a lower minimum when abundant solar production is forecast.

The feature is disabled by default. The default SoC values must be decided before implementation because they affect charging and mobility expectations. A possible initial mapping is Low 80%, Medium 50%, and High 20%.

## Goals

- Derive a vehicle minimum SoC from the normalized evcc solar forecast.
- Support Open-Meteo, Forecast.Solar, Solcast, and other solar tariff providers.
- Make forecast thresholds and per-state minimum SoC values configurable.
- Expose the settings from the solar tariff configuration blade.
- Preserve the existing manually configured vehicle minimum SoC.
- Handle incomplete or unavailable forecasts conservatively.

## Non-goals

- Changing the solar forecast provider format or parsing provider-specific weather responses.
- Replacing charging plans, charge limits, or the existing minimum SoC setting.
- Predicting vehicle usage or required trip energy.
- Dynamically changing the vehicle charge limit.

## User Experience

The solar tariff configuration blade, for example:

`/config?tariff[type:solar]=16`

receives a new **Forecast-based vehicle minimum charge** section.

The section contains:

- An enable toggle.
- A Low/Medium boundary in kWh, initially 5 kWh.
- A Medium/High boundary in kWh, initially 15 kWh.
- One row per SoC-capable vehicle.
- A minimum SoC selector for the Low, Medium, and High states.
- The current 72-hour forecast total.
- The currently selected forecast state.
- The resulting effective minimum SoC.

The thresholds are site-wide because all vehicles consume the same solar forecast. The SoC mappings are per vehicle because mobility requirements differ between vehicles.

The vehicle settings modal continues to edit the manual minimum SoC. It additionally displays the active forecast-derived minimum and links to the solar configuration blade when dynamic control is enabled.

All new user-facing text must be added to `i18n/en.json` and `i18n/de.json`.

## Configuration Ownership

The policy must not be stored inside the Open-Meteo or Forecast.Solar template parameters. Tariff templates configure forecast providers; they should not own vehicle charging policy.

Keeping the settings separate has these advantages:

- Switching forecast providers does not remove the policy.
- The feature automatically works with every `TariffUsageSolar` implementation.
- Vehicle settings remain associated with their vehicle.
- Provider templates do not gain dependencies on the control loop or vehicle APIs.

Suggested persisted settings:

```text
solarMinSoc.enabled
solarMinSoc.lowThreshold
solarMinSoc.mediumThreshold
vehicle.<name>.solarMinSoc.low
vehicle.<name>.solarMinSoc.medium
vehicle.<name>.solarMinSoc.high
```

Threshold validation must require:

```text
0 <= lowThreshold < mediumThreshold
```

SoC values must be between 0 and 100. A value of 0 disables the forecast-derived floor for that state.

## Forecast Calculation

Use the normalized rates from the configured `TariffUsageSolar` tariff. Do not inspect or parse provider-specific payloads.

Calculate forecast energy over the rolling interval from the current time through the next 72 hours:

```text
forecastEnergy = solarEnergy(rates, now, now + 72h) / 1000
```

`solarEnergy` already integrates power samples and returns Wh. Reusing it ensures that partial first and final slots are handled consistently with the existing forecast UI and optimizer.

Classify the result as follows:

```text
forecastEnergy < lowThreshold       => Low
forecastEnergy < mediumThreshold    => Medium
otherwise                           => High
```

The calculation should use the effective adjusted forecast when solar forecast adjustment is enabled. This keeps the policy consistent with the forecast values used for optimization. If applying the adjustment requires a separate helper, it should scale the integrated energy once rather than mutate tariff rates.

## Effective Minimum SoC

The manually configured minimum SoC remains a hard baseline and must never be overwritten by the forecast policy.

For a connected vehicle:

$$
\text{effectiveMinSoc} = \max(\text{loadpointMinSoc}, \text{vehicleMinSoc}, \text{forecastMinSoc})
$$

The existing persisted `vehicleMinSoc` remains user-controlled. `forecastMinSoc` is transient and derived from the latest valid forecast plus the persisted state mapping.

This requires extending the effective minimum SoC calculation in `core/loadpoint_effective.go`. The dynamic value should be available through the vehicle settings adapter or a focused policy interface rather than written into `GetMinSoc()`.

## Missing and Incomplete Forecasts

A missing forecast must not be interpreted as zero production because that would incorrectly select the Low state.

Use these rules:

1. A forecast is valid only if its rates cover the complete rolling 72-hour interval within the normal tariff slot tolerance.
2. While running, retain the last valid state if a later refresh is incomplete or fails.
3. After startup, use only the manual minimum SoC until the first complete forecast is available.
4. Publish availability separately from the selected state so the UI can explain why no dynamic value is active.
5. Log state transitions and availability changes at debug level. Avoid logging on every control-loop iteration.

The retained state is in-memory. Persisting the last calculated state is unnecessary because stale forecasts should not control a fresh process before forecast validity has been established.

## Backend Changes

### Policy model

Add a small policy type responsible for:

- Validating thresholds and per-vehicle mappings.
- Integrating and classifying the forecast.
- Retaining the last valid state.
- Returning the dynamic minimum SoC for a vehicle.

Keep the state names typed, for example `low`, `medium`, and `high`, instead of passing arbitrary strings through the control loop.

### Vehicle settings

Extend the vehicle settings API with methods for reading and updating the three forecast minimum SoC values. Do not change the meaning of `GetMinSoc` or `SetMinSoc`.

Publish the following per vehicle:

- Configured Low, Medium, and High minimum SoC.
- Active forecast-derived minimum SoC.
- Effective minimum SoC.

### Site integration

The site owns the solar tariff and should evaluate the policy once per tariff refresh or when policy settings change. The result must be available before loadpoint updates use `effectiveMinSoc`.

The evaluation should not run from `publishTariffs`, which is primarily presentation and persistence logic. Introduce an explicit update step in the site control path or cache the result when solar rates are refreshed.

Publish site-level state containing:

- Enabled status.
- Configured thresholds.
- Forecast availability.
- Forecast energy in kWh.
- Current state.
- Timestamp of the last valid evaluation.

### HTTP API

Add authenticated configuration endpoints rather than embedding the policy in tariff device YAML. One possible contract is:

```text
GET  /api/config/solar-min-soc
PUT  /api/config/solar-min-soc
```

Example request and response body:

```json
{
  "enabled": true,
  "lowThreshold": 5,
  "mediumThreshold": 15,
  "vehicles": {
    "car": {
      "low": 60,
      "medium": 40,
      "high": 10
    }
  }
}
```

The update must be validated atomically. Invalid thresholds, unknown vehicles, and SoC values outside 0 through 100 should return HTTP 400 without partially changing settings.

The runtime state may be included in the normal WebSocket state instead of the configuration response so changes remain reactive.

### MQTT and OpenAPI

Expose runtime values through MQTT only if they follow the existing state publication automatically. Configuration setters do not need MQTT support in the first version.

Document the new HTTP contract and state fields in the server OpenAPI specifications. Regenerate derived OpenAPI artifacts rather than manually editing generated files where applicable.

## Frontend Changes

### Solar tariff blade

Extend `TariffModal.vue` with a section shown when editing a solar tariff. The section should call the dedicated policy API and remain independent of the selected template.

The section should be visually separated from provider parameters and explain that it controls vehicle charging based on the configured forecast. It should remain visible when the provider is changed.

Because a site may contain multiple solar tariff devices, the policy applies to the combined solar forecast exposed as `TariffUsageSolar`, not to tariff ID 16 specifically.

### Vehicle settings

Extend the vehicle state TypeScript interface with the configured and active dynamic values. In the vehicle settings modal:

- Keep the existing manual minimum SoC control unchanged.
- Show the forecast-derived minimum when enabled.
- Show the effective minimum SoC.
- Link to the solar tariff configuration section.

### State display

The loadpoint status should continue to use `effectiveMinSoc`. No new charging status is required for the initial implementation, but the existing minimum SoC description may mention that the effective value can include a forecast policy.

## Suggested Implementation Sequence

1. Add and test the forecast classification and 72-hour coverage logic.
2. Add persisted site thresholds and per-vehicle state mappings.
3. Publish policy configuration and runtime state.
4. Include the dynamic floor in `effectiveMinSoc`.
5. Add authenticated configuration endpoints and OpenAPI definitions.
6. Add the solar tariff blade controls and vehicle settings display.
7. Add English and German translations.
8. Add integration tests for persistence and the configuration workflow.

## Tests

### Go unit tests

Cover:

- 4.99 kWh selects Low.
- 5 kWh selects Medium.
- 14.99 kWh selects Medium.
- 15 kWh selects High.
- Custom threshold values.
- Partial slots at both ends of the rolling interval.
- Exactly 72 hours of coverage.
- Missing, gapped, and short forecasts.
- Retention of the last valid state after a refresh failure.
- No dynamic minimum after startup without a valid forecast.
- Manual vehicle minimum overriding a lower dynamic value.
- Loadpoint minimum overriding both vehicle values.
- Dynamic minimum overriding lower manual values.
- Separate mappings for multiple vehicles.
- Disabled policy returning no dynamic floor.
- Threshold and SoC validation failures.

### Frontend tests

Cover:

- Loading and updating policy values in the solar tariff blade.
- Validation messages for invalid thresholds.
- Rendering multiple vehicles.
- Displaying forecast availability and current state.
- Showing manual, forecast-derived, and effective minimum SoC values.
- German labels fitting the controls.

### Playwright integration test

Add a solar forecast test configuration and verify this workflow:

1. Open the solar tariff blade.
2. Enable forecast-based minimum SoC.
3. Configure thresholds and vehicle mappings.
4. Save and reopen the blade.
5. Confirm that values persist.
6. Confirm that the effective vehicle minimum changes for a known demo forecast.
7. Restart evcc and confirm the persisted configuration is restored.

## Compatibility and Migration

The feature is additive and disabled by default. Existing installations retain their current minimum SoC behavior.

No migration of existing `minSoc` values is required. Existing API, MQTT, and UI controls keep their current meaning.

Removing or disabling the solar tariff makes the dynamic value unavailable and immediately falls back to the existing manual minimum SoC behavior.

## Open Decisions

- Confirm the initial Low, Medium, and High SoC defaults. Low 80%, Medium 50%, and High 20% are examples only.
- Decide whether the threshold interval should remain fixed at 72 hours in the first version or be configurable later.
- Confirm whether adjusted solar forecasts should drive the policy. The recommended behavior is to use adjustment when enabled.
- Decide whether the policy UI belongs directly inside every solar tariff modal or in a dedicated nested modal linked from that blade. A nested modal may keep provider configuration simpler when many vehicles exist.
