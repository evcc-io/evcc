# Home Assistant Plugins for EVCC

This document describes the new Home Assistant meter and charger plugins that simplify configuration when integrating EVCC with Home Assistant.

## Overview

The Home Assistant plugins eliminate the need for repetitive HTTP configurations by providing dedicated plugin types that handle authentication, entity state management, and service calls automatically.

## Features

### Home Assistant Meter Plugin

- **Type**: `homeassistant`
- **Interfaces**: `api.Meter`, `api.MeterEnergy`, `api.PhaseCurrents`, `api.PhaseVoltages`
- **Purpose**: Read power, energy, current, and voltage values from Home Assistant sensors

### Home Assistant Charger Plugin

- **Type**: `homeassistant`
- **Interfaces**: `api.Charger`, `api.Meter`, `api.MeterEnergy`, `api.PhaseCurrents`, `api.PhaseVoltages`, `api.CurrentGetter`
- **Purpose**: Control charging stations through Home Assistant switches and read their status

## Configuration

### Meter Configuration

```yaml
meters:
  - name: grid_meter
    type: homeassistant
    baseurl: http://homeassistant:8123      # Required: Home Assistant base URL
    token: eyJ...                           # Required: Long-lived access token
    power: sensor.grid_power                # Required: Power sensor entity ID
    energy: sensor.grid_energy              # Optional: Energy sensor entity ID
    currents:                               # Optional: Current sensors for L1, L2, L3
      - sensor.grid_current_l1
      - sensor.grid_current_l2
      - sensor.grid_current_l3
    voltages:                               # Optional: Voltage sensors for L1, L2, L3
      - sensor.grid_voltage_l1
      - sensor.grid_voltage_l2
      - sensor.grid_voltage_l3
```

### Charger Configuration

```yaml
chargers:
  - name: wallbox
    type: homeassistant
    baseurl: http://homeassistant:8123      # Required: Home Assistant base URL
    token: eyJ...                           # Required: Long-lived access token
    status: sensor.wallbox_status           # Required: Status sensor entity ID
    enabled: binary_sensor.wallbox_enabled # Required: Enabled state sensor entity ID
    enable: switch.wallbox_enable           # Required: Enable/disable switch entity ID
    power: sensor.wallbox_power             # Optional: Power sensor entity ID
    energy: sensor.wallbox_energy           # Optional: Energy sensor entity ID
    currents:                               # Optional: Current sensors for L1, L2, L3
      - sensor.wallbox_current_l1
      - sensor.wallbox_current_l2
      - sensor.wallbox_current_l3
    voltages:                               # Optional: Voltage sensors for L1, L2, L3
      - sensor.wallbox_voltage_l1
      - sensor.wallbox_voltage_l2
      - sensor.wallbox_voltage_l3
    maxcurrent: number.wallbox_max_current  # Optional: Max current setting entity ID
```

## Status Mapping

The charger plugin automatically maps Home Assistant states to EVCC charge statuses:

### Status C (Charging)
- `c`, `charging`, `on`, `true`, `active`, `1`

### Status B (Connected/Ready)
- `b`, `connected`, `ready`, `plugged`, `charging_completed`, `initialising`, `preparing`, `2`

### Status A (Disconnected)
- `a`, `disconnected`, `off`, `none`, `unavailable`, `unknown`, `notreadyforcharging`, `not_plugged`, `0`

## Implementation Details

### Shared Connection Utility (`util/homeassistant/connection.go`)

Provides common functionality:
- Bearer token authentication
- Entity state retrieval with error handling
- Service calls (switches, number entities)
- Type conversion helpers (float, bool)
- Charge status mapping
- Three-phase value retrieval

### Error Handling

- Returns `api.ErrNotAvailable` for `unknown` or `unavailable` entity states
- Proper error propagation with context
- Automatic retry logic for transient failures

### Optional Features

Both plugins implement optional interfaces:
- **Energy metering**: Implement if energy sensor is configured
- **Phase measurements**: Implement if current/voltage sensors are configured
- **Phase voltages**: Implement if voltage sensors are configured
- **Current control**: Implement if max current entity is configured (charger only)

## Benefits

1. **Simplified Configuration**: No need for complex HTTP configurations
2. **Automatic Authentication**: Built-in Bearer token handling
3. **Error Handling**: Proper handling of unavailable entities
4. **Type Safety**: Automatic type conversion and validation
5. **Consistency**: Follows existing EVCC plugin patterns
6. **Extensibility**: Easy to add new features and entity types

## Comparison with HTTP Plugin

### Before (HTTP Plugin)
```yaml
meters:
  - name: grid
    type: custom
    power:
      source: http
      uri: http://homeassistant:8123/api/states/sensor.grid_power
      method: GET
      headers:
        - Authorization: Bearer eyJ...
        - Content-Type: application/json
      jq: ".state | tonumber"
    energy:
      source: http
      uri: http://homeassistant:8123/api/states/sensor.grid_energy
      method: GET
      headers:
        - Authorization: Bearer eyJ...
        - Content-Type: application/json
      jq: ".state | tonumber"
    # ... repeat for each sensor
```

### After (Home Assistant Plugin)
```yaml
meters:
  - name: grid
    type: homeassistant
    baseurl: http://homeassistant:8123
    token: eyJ...
    power: sensor.grid_power
    energy: sensor.grid_energy
```

The new plugins reduce configuration complexity by ~80% while providing better error handling and type safety.