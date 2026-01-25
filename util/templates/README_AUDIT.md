# Template Parameter Audit Tool

## Overview

The `audit_required_numeric.py` script audits all template YAML files to identify numeric parameters (`int`, `float`) that are marked as `required: true`. This helps identify parameters that may be affected by zero-value validation issues.

## Background

Related to [Issue #26982](https://github.com/evcc-io/evcc/issues/26982) and [PR #26959](https://github.com/evcc-io/evcc/pull/26959).

The validation logic treats numeric zero values as "empty" for required parameters, which causes validation failures when users explicitly set a required numeric parameter to `0`, even though `0` may be a valid value.

## Usage

### Basic Usage

```bash
cd util/templates
python3 audit_required_numeric.py
```

### Verbose Mode

For more detailed output including parameter descriptions and help text:

```bash
python3 audit_required_numeric.py --verbose
```

## Output

The script produces a report grouped by zero-value validity:

- **YES**: Zero is a valid and semantically meaningful value
- **NO**: Zero is invalid for this parameter
- **MAYBE**: Zero validity depends on device or context
- **UNKNOWN**: Unable to determine validity automatically

### Example Output

```
====================================================================================================
REQUIRED NUMERIC TEMPLATE PARAMETERS AUDIT
====================================================================================================

YES: Zero is valid
----------------------------------------------------------------------------------------------------

  1. Template: open-meteo
     Parameter: az
     Type: int
     Default: None
     File: tariff/open-meteo.yaml

MAYBE: Zero is context-dependent
----------------------------------------------------------------------------------------------------

  1. Template: homematic
     Parameter: meterchannel
     Type: int
     Default: 6
     Example: HMIP-PSM=6, HMIP-FSM+HMIP-FSM16=5, HM=2
     File: charger/homematic.yaml

====================================================================================================
Total: 5 required numeric parameters found

Summary by Zero Value Validity:
  YES: 2
  MAYBE: 3
====================================================================================================
```

## Integration

This script can be integrated into:
- CI/CD pipelines to detect new required numeric parameters
- Pre-commit hooks to validate template changes
- Documentation generation workflows

## Adding New Heuristics

To improve zero-value validity detection, edit the `analyze_parameter()` function in the script. Current heuristics include:

- **Azimuth parameters**: Zero = South orientation (valid)
- **Channel parameters**: Zero = Channel 0 (context-dependent)
- **Damping parameters**: Zero = No damping (valid)
- **Efficiency parameters**: Zero = 0% efficiency (invalid)
- **Power parameters**: Zero = No power/limit (context-dependent)

## Related Documentation

See [REQUIRED_NUMERIC_TEMPLATE_PARAMETERS.md](../../REQUIRED_NUMERIC_TEMPLATE_PARAMETERS.md) for a comprehensive list of all required numeric parameters and their analysis.
