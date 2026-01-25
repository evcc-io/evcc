# Required Numeric Template Parameters

This document lists all numeric template parameters (types `int`, `float`) that are marked as `required: true` in evcc templates.

## Background

Issue: https://github.com/evcc-io/evcc/issues/26982  
Related PR: https://github.com/evcc-io/evcc/pull/26959

The validation logic in `util/templates/template.go` (line 390) checks if required parameters have zero values:

```go
if p.IsRequired() && p.IsZero(s) && (renderMode == RenderModeUnitTest || renderMode == RenderModeInstance && !testing.Testing()) {
    return nil, nil, fmt.Errorf("missing required `%s`", p.Name)
}
```

The `IsZero()` method in `util/templates/types.go` (lines 273-284) treats numeric zero values as "empty":

```go
func (p *Param) IsZero(s string) bool {
	switch p.Type {
	case TypeInt:
		return cast.ToInt64(s) == 0
	case TypeFloat:
		return cast.ToFloat64(s) == 0
	case TypeDuration:
		return cast.ToDuration(s) == 0
	default:
		return len(s) == 0
	}
}
```

This causes validation to fail when a user explicitly sets a required numeric parameter to `0`, even though `0` is a valid value (e.g., `az: 0` for south-facing solar panels in the open-meteo template).

**Note**: The user report mentions `de: 0` (damping evening), but this parameter is NOT marked as `required: true` in the template - it has `default: 0` and `advanced: true`. The actual issue affects the `az` parameter which IS required.

## Complete List of Required Numeric Parameters

### 1. Charger Templates

#### homematic
- **File**: `templates/definition/charger/homematic.yaml`
- **Template**: `homematic`

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `meterchannel` | `int` | `6` | Meter channel number |
| `switchchannel` | `int` | `3` | Switch/Actor channel number |

### 2. Meter Templates

#### homematic
- **File**: `templates/definition/meter/homematic.yaml`
- **Template**: `homematic`

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `meterchannel` | `int` | `6` | Meter channel number |

### 3. Tariff Templates

#### api.akkudoktor.net
- **File**: `templates/definition/tariff/api-akkudoktor-de.yaml`
- **Template**: `api.akkudoktor.net`
- **Status**: `deprecated: true`

| Parameter | Type | Example | Description |
|-----------|------|---------|-------------|
| `az` | `int` | `0` | Azimuth (orientation of PV modules) |

**Note**: Azimuth value of `0` represents south orientation, which is a common and valid configuration for solar panels in the northern hemisphere.

#### open-meteo
- **File**: `templates/definition/tariff/open-meteo.yaml`
- **Template**: `open-meteo`

| Parameter | Type | Example | Description |
|-----------|------|---------|-------------|
| `az` | `int` | `0` | Azimuth (orientation) |

**Note**: Azimuth value of `0` represents south orientation.

## Summary

**Total Required Numeric Parameters**: 5

- **Charger templates**: 3 parameters (2 in homematic charger)
- **Meter templates**: 1 parameter (1 in homematic meter)
- **Tariff templates**: 2 parameters (1 in api.akkudoktor.net, 1 in open-meteo)
- **Vehicle templates**: 0 parameters

## Impact Analysis

### Parameters with Default Values
Most of these parameters have default values:
- `meterchannel`: default `6`
- `switchchannel`: default `3`

These are less likely to be affected since users would typically not override them with `0`.

### Parameters with Example Values
The `az` (azimuth) parameters have example values of `0`:
- This is the primary issue reported, as `0` is a valid and common value for south-facing solar panels
- Users explicitly setting `az: 0` will encounter validation errors

### Parameters Without Defaults
The `az` parameters in both tariff templates (open-meteo and api.akkudoktor.net) do NOT have defaults:
- Users MUST provide a value
- If users set `az: 0`, validation will fail despite `0` being a valid value
- This is particularly problematic since `0` (south) is the most common orientation for solar panels in the northern hemisphere

## Detailed Parameter Analysis

### Parameters Where Zero IS Valid

1. **`az` (azimuth)** - Both tariff templates
   - **Zero means**: South orientation
   - **Valid range**: -180 to 180 degrees
   - **Impact**: HIGH - Zero is a common and semantically meaningful value
   - **Recommendation**: Must allow zero

### Parameters Where Zero MIGHT Be Valid

2. **`meterchannel`** - Homematic templates
   - **Zero means**: Channel 0 (depends on device)
   - **Valid range**: Device-specific
   - **Impact**: MEDIUM - Unlikely but possible on some devices
   - **Recommendation**: Verify with device documentation

3. **`switchchannel`** - Homematic charger template
   - **Zero means**: Channel 0 (depends on device)
   - **Valid range**: Device-specific
   - **Impact**: MEDIUM - Unlikely but possible on some devices
   - **Recommendation**: Verify with device documentation

## Recommendations

1. **Immediate Fix**: Modify the `IsZero()` check to distinguish between "not provided" and "explicitly set to zero"
2. **Consider**: Whether `required: true` is appropriate for parameters with default values (homematic channels)
3. **Review**: Each parameter to determine if zero is a valid value and adjust validation accordingly
4. **Testing**: Add tests to ensure zero values are properly handled for all numeric parameters

## Notes

- All parameters listed here have `required: true` AND `type: int` or `type: float`
- Parameters with `required: true` but type `string` are not affected by this issue
- Parameters with numeric types but without `required: true` are not affected
- The issue primarily affects `az` parameters where zero is both valid and common
