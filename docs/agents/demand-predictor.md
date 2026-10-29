# Heating Demand Predictor

How evcc forecasts energy demand for heating loadpoints (heat pumps, electric radiators, floor heating, hot-water boilers, pool heaters) and feeds that forecast into the optimizer. This is the conceptual model and the reasoning behind it. The code in `core/loadpoint_load_predictor.go`, `core/site_load_predictor.go`, and `core/metrics/db_profile.go` is the reference for details.

## Why heating needs a separate forecast

The optimizer plans solar surplus and cheap-grid windows hours ahead. For EV charging the energy need is known from the SoC target. For heating devices it is unknown: demand changes with outdoor temperature, time of day, season, and the internal behaviour of the controller. Without a forecast the optimizer either ignores the heating loadpoint entirely — missing solar windows — or falls back to the home base load which does not include loadpoint power, causing it to schedule against an incomplete household profile.

The demand predictor adds an energy forecast per heating loadpoint — covering the optimizer's full planning horizon, typically up to 48 hours depending on tariff data availability — on top of the home base load before the optimizer sees it.

## Three forecast strategies

Configured in two distinct places with different value sets:

- **Charger template** (`charger/embed.go`): `predictor: [demandtemperature]` or `predictor: [demandweekday]` — sets `api.DemandTemperature` / `api.DemandWeekday` feature flags that become the device default.
- **Loadpoint** (`evcc.yaml` or UI "Demand forecast" modal): `demandPredictor: temperature` / `weekday` / `daily` — string values, overrides the charger default per loadpoint.

| Strategy | Feature flag | Lookback | Best for |
|---|---|---|---|
| `daily` | *(none — default for any `Heating` loadpoint)* | 28-day all-days average | Any heating device, zero config |
| `weekday` | `api.DemandWeekday` | 28 days, same weekday | Schedule-driven loads: pool heaters, hot tubs, fixed-time boilers |
| `temperature` | `api.DemandTemperature` | 7-day baseline + 1-year binned | Room heating: heat pumps, electric radiators, floor heating |

`activeDemandPredictor()` in `loadpoint_load_predictor.go` resolves the effective strategy: loadpoint-level override first, then charger feature flags, then `daily`.

A prior `DemandDaily` feature flag was removed; the daily average is now the implicit default for any `Heating` loadpoint. Heat pumps without an explicit predictor previously fell outside the home profile entirely, causing the optimizer to plan against a household load that was missing heating consumption.

## Building physics model (temperature strategy)

The temperature strategy begins with a degree-day scaling factor derived from **DIN EN 12831** / **ASHRAE Handbook of Fundamentals**: a building's heat loss is proportional to the indoor–outdoor temperature difference, so electrical input power scales with that difference.

```
                  T_room − T_forecast[slot]
Forecast[slot] = ──────────────────────────── × Load_avg[slot]
                  T_room − T_hist_avg[hour]
```

- `T_room = 21.0 °C` (assumed indoor setpoint, hardcoded)
- `T_forecast` — weather forecast for the slot, from the `TariffUsageTemperature` tariff source (the same open-meteo source used for tariff optimisation)
- `T_hist_avg` — average of all recorded outdoor temperatures at the same **UTC** hour (`r.Start.UTC().Hour()`), taken from past tariff slots

Note: `slotInDay` used for the binned lookup is derived from **local** time (`ts.Local().Hour()*60 + ts.Local().Minute()) / 15`). The two lookups are on different axes — `T_hist_avg` is UTC-hourly, binned slots are local-time-based. They agree at UTC offset 0 and diverge by the UTC offset otherwise.
- Factor is clamped to `[0.1 × … 3.0 ×]` to prevent extreme corrections when the baseline is near zero

The 7-day (not 28-day) baseline window is deliberate: it stays responsive to the current season while being stable enough against noise. The 10% clamp floor preserves standby and DHW load on warm days when space heating has cut off.

## Why the physics model alone is not enough

Three failure modes motivate a data-driven second layer:

**Seasonal transition collapse.** A warm spell drags the 7-day average close to zero. Multiplying a near-zero baseline by a large temperature factor still produces a near-zero forecast — wrong on the first cold day after a warm week. This is the hardest case and the primary motivation for the binned model (PR #34647).

**Non-linear COP.** A heat pump's coefficient of performance degrades non-linearly: roughly 4.5 at +10 °C, 3.0 at +2 °C, 2.0 at −10 °C. Electrical draw therefore scales non-linearly with thermal demand, which the linear formula cannot model.

**Controller non-linearities.** Real heat-pump controllers exhibit:
- Defrost surges: substantial electrical spikes when the evaporator ices over, typically in the −2 °C to +4 °C window
- Morning boost curves: short high-power ramps after night setback
- Compressor modulation limits: step-function behaviour at minimum/maximum load
- DHW priority runs: scheduled high-draw periods mixed into the space heating signal
- Building heating cutoff: zero draw above ~16–18 °C outdoor regardless of the formula

None of these can be captured by a single linear coefficient.

## Temperature-binned self-learning model

`energyProfileTemperatureBinned()` in `core/metrics/db_profile.go` queries SQLite for a 2D lookup table:

```
bins[slot_index][temperature_°C]  →  Wh consumed (average or percentile)
```

- **`slot_index`**: 0–95, one per 15-minute interval across 24 hours, derived from local time
- **`temperature_°C`**: integer, `CAST(round(t.temperature) AS INTEGER)` from the tariff table
- **Lookback**: 1 year rolling (`now.BeginningOfDay().AddDate(-1, 0, 0)`)
- **Minimum samples**: bins with fewer than 2 observations are excluded (`HAVING count(*) >= 2`)
- **Percentile**: configurable via `keys.ProfilePercentile` (0 = average); uses linear interpolation between ranks

This is conceptually a **nearest-neighbour (kNN) lookup**: for a future slot at forecast temperature T, `lookupBin()` in `site_load_predictor.go` searches for an exact integer bin, then falls back to interpolating adjacent ±1 °C bins.

Because the lookup indexes by both time-of-day and temperature it automatically captures everything the physics model cannot — without sub-metering, manual configuration, or model parameters:

- Non-linear COP degradation — actual electricity draw at each temperature is recorded, not modelled
- Defrost cycle surges — the extra power in the −2 °C to +4 °C window appears in exactly those bins
- Combined space heating + DHW — both loads are recorded together; no disaggregation needed
- Scheduled DHW runs — the time-of-day axis captures fixed-schedule high-draw periods regardless of temperature
- Morning boost curves and night setbacks — visible as elevated energy in early-morning slot indices
- Building heating cutoff — bins accumulate near-zero energy above the outdoor cutoff, producing an accurate zero forecast
- Compressor modulation step-functions — reflected in the recorded per-bin averages
- Heating curve adjustments — re-tuning the heat pump's curve is reflected in the same bins within days

## Priority cascade

`applyTemperatureCorrection()` in `site_load_predictor.go` resolves each 15-minute forecast slot in order:

1. `lookupBin(binned[slotInDay], tFuture)` — exact bin match → use recorded average
2. `lookupBin` adjacent ±1 °C interpolation → neighbour exists → interpolate
3. Degree-day scaling × 7-day baseline → historical temperature data available → physics formula
4. Profile slot left unscaled → no temperature data yet (e.g. fresh install, weather source just added)

If `EnergyProfileTemperatureBinned()` returns `ErrNoBinnedData` (normal on first run), `binned` is `nil`. A nil map lookup in `lookupBin` returns `(0, false)` — all slots fall through cleanly to steps 3–4. This is expected, not an error condition.

On a fresh installation the system operates entirely on the physics model. As it runs through its first season, bins accumulate for the temperatures seen. After a full year the entire operating range is covered and the physics fallback is almost never triggered. Each slot independently promotes from physics to data-driven as its bin reaches the 2-sample threshold — no retraining, no configuration change.

Seasonal transitions are handled naturally: the 1-year lookback carries observations from the same calendar period a year ago, so a cold snap after a warm week finds matching low-temperature bins from the previous winter rather than extrapolating from the near-zero recent baseline.

## Integration with the optimizer

`addHeatingDemand()` in `site_load_predictor.go` iterates all loadpoints before the optimizer runs:

- Skips loadpoints where status is not `StatusB` (connected, not charging) or `StatusC` (charging) — i.e. disconnected or unknown
- Skips non-`Continuous` loadpoints in `ModeOff`; `Continuous` devices in `ModeOff` mean "normal operation" and are included
- Calls `lp.demandProfile()` for daily/temperature strategies, `lp.demandProfileWeekday()` for weekday
- Temperature correction (`applyTemperatureCorrection`) is called when `demandProfile()` returns `correct = true`, even if binned data is not yet available (nil map falls through to physics)
- The resulting Wh values are added directly to the `gt` (grid total) forecast slice

The optimizer therefore sees a home load that already includes all active heating loadpoints, giving it an accurate demand baseline to plan solar surplus against.

## Key files

| File | Purpose |
|---|---|
| `api/feature.go` | `DemandWeekday`, `DemandTemperature` feature flag constants |
| `charger/embed.go` | `predictor:` YAML key → feature flags mapping for charger templates |
| `core/loadpoint_load_predictor.go` | Strategy resolution, `GetDemandPredictor()`, `SetDemandPredictor()`, `demandProfile()`, `demandProfileWeekday()` |
| `core/site_load_predictor.go` | `addHeatingDemand()`, `applyTemperatureCorrection()`, `lookupBin()` |
| `core/metrics/db_profile.go` | `energyProfileTemperatureBinned()`, `energyProfileFiltered()` SQL queries |
| `server/http_config_loadpoint_handler.go` | Exposes `demandPredictor` in the loadpoint config API response |
| `assets/js/components/Config/LoadpointModal.vue` | UI "Demand forecast" selector, warning for missing weather source |

## Pull requests

- **#28232** — initial weekday and temperature strategies; YAML opt-in
- **#34342** — UI demand forecast selector in loadpoint settings modal
- **#34647** — temperature-binned self-learning model; seasonal transition fix; linear scaling fallback
