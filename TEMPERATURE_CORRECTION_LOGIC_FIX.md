# Temperature Correction Logic Fix

## Problem Statement

The current implementation has a fallback behavior that is incorrect:

```go
// If no heating devices or no heater data, use old behavior (apply correction to entire profile)
if gt_heater_raw == nil || len(gt_heater_raw) == 0 {
    res = site.applyTemperatureCorrection(res)
    // convert to Wh
    return lo.Map(res, func(v float64, i int) float64 {
        return v * 1e3
    }), nil
}
```

**This is wrong** because:
1. The "old behavior" (applying temperature correction to entire household load) should **never** be used
2. Temperature correction should **only** apply to heating device loads
3. If there are no heating devices or no heater data, **no correction should happen at all**

## Current Behavior Analysis

### When `applyTemperatureCorrection()` is called:

The function already has proper safeguards:
- Returns uncorrected profile if no weather tariff configured (line 607-609)
- Returns uncorrected profile if no weather rates available (line 611-613)
- Returns uncorrected profile if past 24h avg temp >= heating threshold (line 643-645)
- Returns uncorrected profile if no past temperature data for a given hour (line 638-641)

### The Issue in `homeProfile()`:

Lines 537-543 incorrectly apply temperature correction to the **entire household profile** when:
- No heating devices are configured (`gt_heater_raw == nil`)
- No heater consumption data available (`len(gt_heater_raw) == 0`)

This defeats the entire purpose of the heater profile separation feature.

## Correct Behavior

### Scenario 1: No Heating Devices Configured
**Condition**: `gt_heater_raw == nil || len(gt_heater_raw) == 0`
**Action**: Return the **uncorrected** total household profile
**Reason**: Without heating devices, there's nothing to temperature-correct

### Scenario 2: Heating Devices Exist, No Weather Data
**Condition**: Weather tariff not configured or no rates available
**Action**: `applyTemperatureCorrection()` returns uncorrected heater profile
**Result**: Final profile = base_load + uncorrected_heater_load
**Reason**: Can't apply correction without temperature data

### Scenario 3: Heating Devices Exist, Weather Data Available
**Condition**: All data present
**Action**: Apply temperature correction to heater profile only
**Result**: Final profile = base_load + corrected_heater_load
**Reason**: This is the intended behavior

## Required Code Changes

### In `homeProfile()` function (lines 537-543):

**Current (WRONG)**:
```go
// If no heating devices or no heater data, use old behavior (apply correction to entire profile)
if gt_heater_raw == nil || len(gt_heater_raw) == 0 {
    res = site.applyTemperatureCorrection(res)
    // convert to Wh
    return lo.Map(res, func(v float64, i int) float64 {
        return v * 1e3
    }), nil
}
```

**Corrected**:
```go
// If no heating devices or no heater data, return uncorrected profile
// Temperature correction should ONLY apply to heating loads, never to entire household
if gt_heater_raw == nil || len(gt_heater_raw) == 0 {
    // convert to Wh
    return lo.Map(res, func(v float64, i int) float64 {
        return v * 1e3
    }), nil
}
```

## Impact Analysis

### Before Fix:
- ❌ Systems without heating devices: Entire household load incorrectly adjusted for temperature
- ❌ Systems with heating devices but no data: Entire household load incorrectly adjusted
- ✅ Systems with heating devices and data: Only heater load adjusted (correct)

### After Fix:
- ✅ Systems without heating devices: No temperature correction (correct)
- ✅ Systems with heating devices but no data: No temperature correction (correct)
- ✅ Systems with heating devices and data: Only heater load adjusted (correct)

## Testing Scenarios

1. **No heating devices configured**
   - Expected: Profile returned unchanged
   - Verify: No temperature correction applied

2. **Heating devices configured, no historical data yet**
   - Expected: Profile returned unchanged
   - Verify: No temperature correction applied

3. **Heating devices configured, no weather data**
   - Expected: Profile = base + uncorrected heater
   - Verify: `applyTemperatureCorrection()` returns uncorrected heater profile

4. **Heating devices configured, weather data available**
   - Expected: Profile = base + corrected heater
   - Verify: Only heater portion is temperature-adjusted

## Summary

The fix is simple: **Remove the call to `applyTemperatureCorrection(res)`** from the fallback path. The function should simply return the uncorrected profile when heating devices or heater data are not available.

This ensures temperature correction is **only and always** applied to heating device loads, never to the entire household consumption.