# Temperature-Based Household Load Correction with Heater Profile Separation

**Base PR**: #27780 (Weather Tariff)  
**This PR**: Heater Profile Separation for Temperature Correction

## Problem

The evopt energy optimizer uses a **household load profile** (`gt`) to predict how much energy the home will consume in each future 15-minute slot. This profile is currently computed as a **30-day historical average** — the same flat pattern is repeated regardless of weather conditions.

This is a significant blind spot: **heating and cooling loads are strongly temperature-dependent**. On a cold winter day, a home with a heat pump or electric heating can consume 30–80% more energy than on a mild day. When the optimizer doesn't know this, it:

- Under-estimates household demand on cold days → schedules EV charging at times when grid power is actually needed for heating
- Over-estimates household demand on warm days → unnecessarily avoids cheap/green charging windows
- Misses opportunities to pre-charge the battery before a cold night when heating demand will spike

## Solution Overview

This PR builds on #27780 (which added the `open-meteo` weather tariff) and implements **temperature-based correction of the household load profile** with a critical improvement: **the correction is applied only to heating device loads, not the entire household consumption**.

### Key Innovation: Heater Profile Separation

The original approach (discussed in #27780) applied temperature correction to the entire household profile. However, as correctly identified by the project owner, **only heating/cooling loads are temperature-dependent**. Base household loads (lighting, appliances, cooking, etc.) remain constant regardless of outdoor temperature.

This PR implements a **three-step process**:

1. **Separate**: Extract heater consumption into its own profile (`gt_heater`)
2. **Correct**: Apply temperature adjustment only to `gt_heater`
3. **Merge**: Combine corrected heater profile with unchanged base load

```
Total Household (gt_total) = Base Load (gt_base) + Heater Load (gt_heater)
                                    ↓                        ↓
                              [unchanged]        [temperature corrected]
                                    ↓                        ↓
                          Final Profile = gt_base + gt_heater_corrected
```

## Implementation Details

### 1. Per-Loadpoint Metrics Tracking

**Files**: `core/metrics/db.go`, `core/site.go`

- Extended metrics database to track consumption per loadpoint
- Added `PersistLoadpoint()` and `LoadpointProfile()` functions
- Heating devices identified via `api.Heating` feature flag
- Mirrors existing household metrics pattern (15-minute slots)

```go
// Site struct additions
loadpointEnergy    map[int]*meterEnergy
loadpointSlotStart map[int]time.Time

// Initialization in Boot()
for i, lp := range loadpoints {
    if hasFeature(lp.charger, api.Heating) {
        site.loadpointEnergy[i] = &meterEnergy{clock: clock.New()}
    }
}
```

### 2. Profile Extraction and Aggregation

**File**: `core/site_optimizer.go`

New helper functions:
- `getHeatingLoadpoints()`: Identifies all heating devices
- `extractHeaterProfile()`: Queries historical heater consumption
- `sumProfiles()`: Aggregates multiple heating devices

### 3. Modified Temperature Correction Flow

**File**: `core/site_optimizer.go` - `homeProfile()` function

```go
// Query both profiles
gt_total := metrics.Profile(from)           // Total household
gt_heater := extractHeaterProfile(from, to) // Heaters only

// Calculate base load (non-heating)
gt_base[i] = gt_total[i] - gt_heater[i]

// Apply temperature correction ONLY to heater profile
gt_heater_corrected := applyTemperatureCorrection(gt_heater)

// Merge back
gt_final[i] = gt_base[i] + gt_heater_corrected[i]
```

### Temperature Correction Algorithm

The correction algorithm (from base PR #27780) remains unchanged:

```
load[i] = load_avg[i] × (1 + heatingCoefficient × (T_past_avg[h] − T_forecast[i]))
```

where:
- `T_past_avg[h]` = average temperature at hour-of-day `h` over the past 7 days
- `T_forecast[i]` = forecast temperature at the wall-clock time of slot `i`
- `heatingCoefficient` = fractional load sensitivity per °C (default 0.05 = 5%/°C)

**The correction is gated on the 24h average actual temperature of the past 24 hours.** If that average is at or above `heatingThreshold` (default 12°C), heating is considered off and no correction is applied to any slot.

**Example:** With default settings and a 7-day historical average of 8°C:
- Forecast 8°C → no correction (delta = 0, factor = 1.0)
- Forecast 3°C → +25% heater load (delta = +5°C, factor = 1.25)
- Forecast −2°C → +50% heater load (delta = +10°C, factor = 1.50)
- Forecast 13°C → −25% heater load (delta = −5°C, factor = 0.75)

## Files Changed

| File | Change |
|------|--------|
| `core/metrics/db.go` | Extended schema for per-loadpoint tracking (from previous work) |
| `core/site.go` | Added loadpoint energy tracking infrastructure (~50 lines) |
| `core/site_optimizer.go` | Added profile extraction and modified correction flow (~130 lines) |

## Configuration

The feature is **opt-in** — no changes needed for existing setups.

To enable (requires base PR #27780):

```yaml
tariffs:
  weather:
    type: open-meteo
    latitude: 48.137   # your location
    longitude: 11.576

site:
  title: My Home
  meters:
    grid: grid0
    pv: pv0
  # Optional tuning (these are the defaults):
  heatingThreshold: 12.0    # °C — 24h avg above which corrections are disabled
  heatingCoefficient: 0.05  # fraction — load changes by this fraction per °C delta
```

### Tuning Guidance

- **`heatingThreshold`**: Set to the 24h average outdoor temperature above which your heating system is fully off. Typical values: 10–15°C depending on building insulation. Default 12°C suits an average insulated house.
- **`heatingCoefficient`**: Represents how sensitive your home's energy consumption is to temperature. A well-insulated passive house might use 0.02; a poorly insulated older building might use 0.08 or higher.

## Backward Compatibility

- **No heating devices**: Falls back to old behavior (no temperature correction)
- **Heating devices but no weather tariff**: No temperature correction applied
- **Heating devices + weather tariff**: Temperature correction only on heater portion
- **No configuration changes required**: Heating devices automatically detected via `api.Heating` feature flag

## Benefits

### Accuracy Improvements
- **More precise optimizer predictions** on temperature extremes
- **Better EV charging schedules** that don't conflict with heating needs
- **Improved battery utilization** during cold/warm periods
- **Preserved daily patterns** (evening peaks, etc.) while adjusting magnitude

### Code Quality
- **Cleaner separation of concerns** (base vs. temperature-dependent loads)
- **More maintainable** temperature correction logic
- **Extensible** for future enhancements (cooling support, per-device analysis)

## Notes

- Heating devices are automatically identified via existing `api.Heating` feature flag (heat pumps, electric heaters, etc.)
- Historical heater data starts accumulating from deployment — no migration needed
- The correction only applies to **future slots** in the optimizer horizon
- Safe degradation if heater data unavailable (falls back to full-profile correction)
- Open-Meteo is fetched once per hour with exponential backoff on failure
- Past temperatures (7 days) and future forecast (3 days) fetched in **single API call**
- The 24h average gate uses **past 24h actual temperatures** (not forecast) for reliability

## Future Enhancements

1. **Cooling support**: Add `coolingThreshold` / `coolingCoefficient` for summer A/C
2. **Per-device profiles**: Track each heating device separately for detailed analysis
3. **Auto-calibration**: Estimate `heatingCoefficient` from historical consumption patterns
4. **UI visualization**: Show base vs. heater load breakdown in dashboard

## Testing

- Code review validation completed
- CI/CD will validate: compilation, unit tests, linter, integration tests
- Manual testing recommended with real heat pump installation

## Dependencies

- Requires base PR #27780 (Weather Tariff) to be merged first
- No new external dependencies
- Uses existing `api.Heating` feature flag for device identification