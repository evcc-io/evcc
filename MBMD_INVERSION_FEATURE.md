# MBMD Meters: Value Negation Feature

## Overview

This feature adds support for inverting measurement values in mbmd meters by prefixing alias names with a `-` (minus sign). This is particularly useful when meters are used in different configurations or mounting orientations and the sign of measurement values needs to be adjusted.

## Usage

In `mbmd` type templates, you can now prefix alias names with `-`. The meter will automatically invert the sign of the respective measurement value.

### Syntax

```yaml
type: mbmd
model: <model-name>
power: -Power      # inverts the power value
```

**Note**: Negation is **only** available for `power` and phase arrays (`currents`, `powers`). Negation is not possible for `energy`, `soc`, and `voltages` as these values should always be positive.

### Examples

#### Example 1: Inverted Total Power, Exported Energy

```yaml
type: mbmd
model: cgex3x0
power: -Power      # Power will be inverted (e.g. +1000W becomes -1000W)
energy: Export     # Energy CANNOT be inverted
```

#### Example 2: Inverted Phase Powers

```yaml
type: mbmd
model: cgex3x0
power: Power
powers:
  - -PowerL1      # Phase 1 inverted
  - -PowerL2      # Phase 2 inverted
  - -PowerL3      # Phase 3 inverted
```

#### Example 3: Inverted Currents (very theoretical)

```yaml
type: mbmd
model: cgex3x0
power: Power
currents:
  - -CurrentL1    # Current Phase 1 inverted
  - CurrentL2     # Current Phase 2 normal
  - -CurrentL3    # Current Phase 3 inverted
```

## Technical Details

### Supported Measurements

Negation works for the following mbmd measurements:

- `power` - Total power ✓
- `currents` - Currents per phase (array) ✓
- `powers` - Powers per phase (array) ✓

**Not supported** (and not sensible):
- `energy` - Energy (Import/Export) ✗ - Energy counter values are always positive
- `soc` - State of Charge (for batteries) ✗ - SOC values are always positive (0-100%)
- `voltages` - Voltages per phase (array) ✗ - Voltage values are always positive (e.g. 230V)

### Behavior

- **Normal operation**: `power: Power` → Value is returned unchanged
- **Inverted**: `power: -Power` → Value is multiplied by -1
- **NaN error handling**: Remains intact (NaN becomes 0, then optionally inverted)

## Use Cases

### 1. Incorrectly Mounted Meters

If a current sensor/meter is physically installed in the "wrong" direction, you can correct the values in software:

```yaml
power: -Power
```

### 2. Grid Feed-in vs. Consumption

For PV meters where the sign convention needs to be reversed:

```yaml
type: mbmd
usage: grid
power: -Power
```

### 3. Different CT Orientations

For multi-phase installations with differently oriented current transformers:

```yaml
currents:
  - CurrentL1      # CT normally oriented
  - -CurrentL2     # CT reversed
  - CurrentL3      # CT normally oriented
```