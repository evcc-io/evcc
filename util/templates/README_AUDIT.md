# Template Parameter Audit

## Overview

This document describes how to audit template YAML files to identify numeric parameters (`int`, `float`) that are marked as `required: true`. This helps identify parameters that may be affected by zero-value validation issues.

## Background

Related to [Issue #26982](https://github.com/evcc-io/evcc/issues/26982) and [PR #26959](https://github.com/evcc-io/evcc/pull/26959).

The validation logic treats numeric zero values as "empty" for required parameters, which causes validation failures when users explicitly set a required numeric parameter to `0`, even though `0` may be a valid value.

## Quick Audit with Shell Commands

### Find all required numeric parameters

```bash
cd templates/definition
grep -r "required: true" --include="*.yaml" | grep -B 5 -E "type: (int|float)" | grep -E "(template:|name:|type:|required:|default:)" | less
```

### Count required numeric parameters

```bash
cd templates/definition
for file in $(find . -name "*.yaml"); do
  yq -r '.params[] | select(.required == true and (.type == "int" or .type == "float")) | "\(.name) (\(.type))"' "$file" 2>/dev/null | while read param; do
    template=$(yq -r '.template' "$file" 2>/dev/null)
    [ -n "$param" ] && echo "$template: $param ($file)"
  done
done
```

## Current Results

As of the last audit, there are **5 required numeric parameters**:

### Parameters Where Zero IS Valid

1. **`az` (azimuth)** in `tariff/api-akkudoktor-de.yaml` and `tariff/open-meteo.yaml`
   - **Type**: `int`
   - **Zero means**: South orientation
   - **Impact**: HIGH - Zero is a common and semantically meaningful value

### Parameters Where Zero MIGHT Be Valid

2. **`meterchannel`** in `charger/homematic.yaml` and `meter/homematic.yaml`
   - **Type**: `int`
   - **Default**: `6`
   - **Zero means**: Channel 0 (depends on device)
   - **Impact**: MEDIUM - Unlikely but possible on some devices

3. **`switchchannel`** in `charger/homematic.yaml`
   - **Type**: `int`
   - **Default**: `3`
   - **Zero means**: Channel 0 (depends on device)
   - **Impact**: MEDIUM - Unlikely but possible on some devices

## Related Documentation

See [REQUIRED_NUMERIC_TEMPLATE_PARAMETERS.md](../../REQUIRED_NUMERIC_TEMPLATE_PARAMETERS.md) for a comprehensive list of all required numeric parameters and detailed analysis.
