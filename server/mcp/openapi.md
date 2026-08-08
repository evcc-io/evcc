# MCP Tools Documentation

**API Title:** evcc

**Version:** 0.2.0

Solar charging. Super simple.

## changePassword

Changes the admin password.

**Tags:** auth

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| requestBody | object | The JSON request body. |

**Example call:**

```json
call changePassword {
  "requestBody": "..."
}
```

## getApiKeyStatus

Reports whether an API key has been generated. The key itself is
only returned once at creation time (POST), never on subsequent
reads.


**Tags:** auth

## getAuthStatus

Whether the current user is logged in.

**Tags:** auth

## login

Administrator login. Returns authorization cookie required for all protected endpoints.

**Tags:** auth

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| requestBody | object | The JSON request body. |

**Example call:**

```json
call login {
  "requestBody": "..."
}
```

## logout

Logout and delete authorization cookie

**Tags:** auth

## regenerateApiKey

Generates a fresh API key, replacing any existing one. The
returned key is shown only once; store it immediately.

Even when the request is authenticated via API key (Bearer), the
admin password must be supplied in the request body to prevent a
leaked key from rotating itself. The password check is skipped
when the server is started with `--disable-auth`.


**Tags:** auth

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| requestBody | object | The JSON request body. |

**Example call:**

```json
call regenerateApiKey {
  "requestBody": "..."
}
```

## disableExternalBatteryControl

Default evcc control behavior is restored

**Tags:** battery

## removeBatteryGridChargeLimit

Remove battery grid charge limit.

**Tags:** battery

## setBatteryDischargeControl

Prevent home battery discharge during vehicle fast charging.

**Tags:** battery

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| enable | string | Charging mode. |

**Example call:**

```json
call setBatteryDischargeControl {
  "enable": "true"
}
```

## setBatteryGridChargeLimit

Charge home battery from grid when price or emissions are below the threshold. Uses price if a dynamic tariff exists. Uses emissions if a CO₂-tariff is configured. Ignored otherwise.

**Tags:** battery

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| cost | number | Cost limit in configured currency (default EUR) or CO2 limit in g/kWh |

**Example call:**

```json
call setBatteryGridChargeLimit {
  "cost": 123.45
}
```

## setBatteryGridDischarge

Allow the home battery to discharge to the grid (experimental).

**Tags:** battery

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| enable | string | Charging mode. |

**Example call:**

```json
call setBatteryGridDischarge {
  "enable": "true"
}
```

## setBufferSoc

Set battery buffer SoC.

**Tags:** battery

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| soc | number | SOC in % |

**Example call:**

```json
call setBufferSoc {
  "soc": 60
}
```

## setBufferStartSoc

Set battery buffer start SoC.

**Tags:** battery

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| soc | number | SOC in % |

**Example call:**

```json
call setBufferStartSoc {
  "soc": 60
}
```

## setExternalBatteryMode

Directly controls the mode of all controllable batteries. evcc behavior like 'price limit' or 'prevent discharge while fast charging' is overruled. External mode resets after 60s. The external system has to call this endpoint regularly.

**Tags:** battery

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| batteryMode | string | Battery mode |

**Example call:**

```json
call setExternalBatteryMode {
  "batteryMode": "normal"
}
```

## setPrioritySoc

Set battery priority SoC.

**Tags:** battery

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| soc | number | SOC in % |

**Example call:**

```json
call setPrioritySoc {
  "soc": 60
}
```

## setResidualPower

Set grid connection operating point.

**Tags:** battery

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| power | number | Power in W |

**Example call:**

```json
call setResidualPower {
  "power": 2500
}
```

## downloadBackup

Downloads the SQLite database as a backup file. Session users must supply the admin password in the X-Admin-Password header. API key holders via Bearer token are exempt.

**Tags:** db

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| X-Admin-Password | string | Admin password (required for session auth, not needed for API key) |

**Example call:**

```json
call downloadBackup {
  "X-Admin-Password": "example"
}
```

## resetDatabase

Selectively deletes sessions and/or settings from the database. Session users must supply the admin password in the X-Admin-Password header. API key holders via Bearer token are exempt. The instance restarts after a successful reset.

**Tags:** db

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| X-Admin-Password | string | Admin password (required for session auth, not needed for API key) |
| requestBody | object | The JSON request body. |

**Example call:**

```json
call resetDatabase {
  "X-Admin-Password": "example",
  "requestBody": "..."
}
```

## restoreBackup

Restores the database from a previously downloaded backup file. Session users must supply the admin password in the X-Admin-Password header. API key holders via Bearer token are exempt. The instance restarts after a successful restore.

**Tags:** db

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| X-Admin-Password | string | Admin password (required for session auth, not needed for API key) |

**Example call:**

```json
call restoreBackup {
  "X-Admin-Password": "example"
}
```

## getEnergyHistory

Returns aggregated energy history data. Aggregate granularity defaults to 15 minutes. Supports CSV export.

**Tags:** experimental

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| aggregate | string | Aggregation interval. Examples: 15m, 1h, day, month |
| format | string | Response format |
| from | string | Start time (RFC3339) |
| group | string | Filter by entity group |
| grouped | boolean | Group results by loadpoint |
| lang | string | Language for CSV column headers (BCP 47, e.g. de, en). Defaults to Accept-Language header. |
| name | string | Filter by entity name |
| title | string | Filter by entity title |
| to | string | End time (RFC3339) |

**Example call:**

```json
call getEnergyHistory {
  "aggregate": "1h",
  "format": "json",
  "from": "2026-07-01T00:00:00Z",
  "group": "battery",
  "grouped": false,
  "lang": "de",
  "name": "db:8",
  "title": "Battery",
  "to": "2026-07-02T00:00:00Z"
}
```

## setSolarAdjusted

Adjust the solar forecast to real production data of the current day.

**Tags:** experimental

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| enable | string | Charging mode. |

**Example call:**

```json
call setSolarAdjusted {
  "enable": "true"
}
```

## getState

Returns the complete state of the system. This structure is used by the UI. It can be filtered by JQ to only return a subset of the data.

**Tags:** general

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| jq | string | Filter the state with JQ |

**Example call:**

```json
call getState {
  "jq": "example"
}
```

## removeGlobalSmartCostLimit

Convenience method to remove limit for all loadpoints at once. Value is applied to each individual loadpoint.

**Tags:** general

## removeGlobalSmartFeedInPriorityLimit

Convenience method to remove limit for all loadpoints at once. Value is applied to each individual loadpoint.

**Tags:** general

## setGlobalSmartCostLimit

Convenience method to set smart charging cost limit for all loadpoints at once. Value is applied to each individual loadpoint.

**Tags:** general

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| cost | number | Cost limit in configured currency (default EUR) or CO2 limit in g/kWh |

**Example call:**

```json
call setGlobalSmartCostLimit {
  "cost": 123.45
}
```

## setGlobalSmartFeedInPriorityLimit

Convenience method to set smart feed-in priority limit for all loadpoints at once. Value is applied to each individual loadpoint.

**Tags:** general

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| cost | number | Cost limit in configured currency (default EUR) or CO2 limit in g/kWh |

**Example call:**

```json
call setGlobalSmartFeedInPriorityLimit {
  "cost": 123.45
}
```

## assignLoadpointVehicle

Assigns vehicle to loadpoint.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| name | string | Vehicle name |

**Example call:**

```json
call assignLoadpointVehicle {
  "id": 1,
  "name": "vehicle_1"
}
```

## deleteLoadpointEnergyPlan

Delete charging plan. Only available when a vehicle without SoC is connected.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call deleteLoadpointEnergyPlan {
  "id": 1
}
```

## deleteLoadpointSmartCostLimit

Delete cost or emission limit for fast-charging with grid energy.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call deleteLoadpointSmartCostLimit {
  "id": 1
}
```

## deleteLoadpointSmartFeedInPriorityLimit

Delete limit for feed-in priority optimization.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call deleteLoadpointSmartFeedInPriorityLimit {
  "id": 1
}
```

## getLoadpointPlan

Returns the current charging plan for this loadpoint.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call getLoadpointPlan {
  "id": 1
}
```

## previewLoadpointEnergyPlan

Simulate charging plan based on energy goal. Does not alter the actual charging plan.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| energy | number | Energy in kWh |
| id | integer | Loadpoint index starting at 1 |
| timestamp | string | Timestamp in RFC3339 format |

**Example call:**

```json
call previewLoadpointEnergyPlan {
  "energy": 25.5,
  "id": 1,
  "timestamp": "example"
}
```

## previewLoadpointRepeatingPlan

Simulate repeating charging plan and return the result. Does not alter the actual charging plan.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| hourMinuteTime | string | Time in `HOURS:MINUTES` format |
| id | integer | Loadpoint index starting at 1 |
| soc | number | SOC in % |
| timezone | string | Timezone in IANA format |
| weekdays | array | The Weekdays |

**Example call:**

```json
call previewLoadpointRepeatingPlan {
  "hourMinuteTime": "12:30",
  "id": 1,
  "soc": 60,
  "timezone": "Europe/Berlin",
  "weekdays": [
    1,
    3,
    4
  ]
}
```

## previewLoadpointSocPlan

Simulate charging plan based on SoC goal. Does not alter the actual charging plan.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| soc | number | SOC in % |
| timestamp | string | Timestamp in RFC3339 format |

**Example call:**

```json
call previewLoadpointSocPlan {
  "id": 1,
  "soc": 60,
  "timestamp": "example"
}
```

## removeLoadpointVehicle

Remove vehicle from loadpoint. Connected vehicle is treated as guest vehicle.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call removeLoadpointVehicle {
  "id": 1
}
```

## setLoadpointBatteryBoost

Enable or disable battery boost. When active, the maximum available home battery power is added until the home battery is drained to configured SoC limit. Note: boost will not work while the battery is on hold (e.g. during fast charging or planned charging with discharge prevention enabled).

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| enable | string | Charging mode. |
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call setLoadpointBatteryBoost {
  "enable": "true",
  "id": 1
}
```

## setLoadpointBatteryBoostLimit

Set the SoC limit for battery boost. Home battery will be used to support charging up to this SoC level. A value of 100 (default) disabled the boost feature in UI and API.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| soc | number | SOC in % |

**Example call:**

```json
call setLoadpointBatteryBoostLimit {
  "id": 1,
  "soc": 60
}
```

## setLoadpointDisableDelay

Delay before charging stops in solar mode.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| delay | integer | Duration in seconds. |
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call setLoadpointDisableDelay {
  "delay": 60,
  "id": 1
}
```

## setLoadpointDisableThreshold

Specifies the grid draw power to stop charging in solar mode.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| threshold | number | Power in W |

**Example call:**

```json
call setLoadpointDisableThreshold {
  "id": 1,
  "threshold": 2500
}
```

## setLoadpointEnableDelay

Delay before charging starts in solar mode.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| delay | integer | Duration in seconds. |
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call setLoadpointEnableDelay {
  "delay": 60,
  "id": 1
}
```

## setLoadpointEnableThreshold

Specifies the available surplus power to start charging in solar mode.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| threshold | number | Power in W |

**Example call:**

```json
call setLoadpointEnableThreshold {
  "id": 1,
  "threshold": 2500
}
```

## setLoadpointEnergyLimit

Updates the energy limit of the loadpoint. Only available for guest vehicles and vehicles with unknown SoC. Limit is removed on vehicle disconnect.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| energy | number | Energy in kWh |
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call setLoadpointEnergyLimit {
  "energy": 25.5,
  "id": 1
}
```

## setLoadpointEnergyPlan

Create charging plan with fixed time and energy target. Only available when a vehicle without SoC is connected.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| energy | number | Energy in kWh |
| id | integer | Loadpoint index starting at 1 |
| timestamp | string | Timestamp in RFC3339 format |

**Example call:**

```json
call setLoadpointEnergyPlan {
  "energy": 25.5,
  "id": 1,
  "timestamp": "example"
}
```

## setLoadpointMaxCurrent

Updates the maximum current of the loadpoint.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| current | number | Electric current in A |
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call setLoadpointMaxCurrent {
  "current": 16,
  "id": 1
}
```

## setLoadpointMinCurrent

Updates the minimum current of the loadpoint.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| current | number | Electric current in A |
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call setLoadpointMinCurrent {
  "current": 16,
  "id": 1
}
```

## setLoadpointMinTemp

Sets the minimum temperature for heating devices. The device is heated as fast as possible while below this value. Set to 0 to disable.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| temp | number | Temperature in °C |

**Example call:**

```json
call setLoadpointMinTemp {
  "id": 1,
  "temp": 50
}
```

## setLoadpointMode

Changes the charging behavior of the loadpoint.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| mode | string | Charging mode. |

**Example call:**

```json
call setLoadpointMode {
  "id": 1,
  "mode": "off"
}
```

## setLoadpointPhases

Updates the allowed phases of the loadpoint. Selects the desired phase mode for chargers with automatic phase switching. For manual phase switching chargers (via cable or Lasttrennschalter) this value tells evcc the actual phases.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| phases | string | Number of phases. (0: auto, 1: 1-phase, 3: 3-phase) |

**Example call:**

```json
call setLoadpointPhases {
  "id": 1,
  "phases": "3"
}
```

## setLoadpointPlanStrategy

Updates the charging plan strategy for the loadpoint.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| requestBody | object | The JSON request body. |

**Example call:**

```json
call setLoadpointPlanStrategy {
  "id": 1,
  "requestBody": "..."
}
```

## setLoadpointPriority

Set loadpoint priority.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| priority | integer | Higher number means higher priority. |

**Example call:**

```json
call setLoadpointPriority {
  "id": 1,
  "priority": 123
}
```

## setLoadpointSmartCostLimit

Set cost or emission limit for fast-charging with grid energy.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| cost | number | Cost limit in configured currency (default EUR) or CO2 limit in g/kWh |
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call setLoadpointSmartCostLimit {
  "cost": 123.45,
  "id": 1
}
```

## setLoadpointSmartFeedInPriorityLimit

Set limit for feed-in priority optimization.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| cost | number | Cost limit in configured currency (default EUR) or CO2 limit in g/kWh |
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call setLoadpointSmartFeedInPriorityLimit {
  "cost": 123.45,
  "id": 1
}
```

## setLoadpointSocLimit

Sets the session SoC limit. Cleared on disconnect. Takes precedence over the vehicle's configured limit while set; once cleared (set to 0), the vehicle limit applies again.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| soc | number | SOC in % |

**Example call:**

```json
call setLoadpointSocLimit {
  "id": 1,
  "soc": 60
}
```

## startLoadpointVehicleDetection

Starts the automatic vehicle detection process.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call startLoadpointVehicleDetection {
  "id": 1
}
```

## deleteSession

Delete charging session.

**Tags:** sessions

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call deleteSession {
  "id": 1
}
```

## getGridSessions

Returns a list of HEMS grid limitation events.

**Tags:** sessions

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| format | string | Response format (default json) |
| lang | string | Language (defaults to accept header) |

**Example call:**

```json
call getGridSessions {
  "format": "csv",
  "lang": "de"
}
```

## getSessions

Returns a list of charging sessions.

**Tags:** sessions

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| format | string | Response format (default json) |
| lang | string | Language (defaults to accept header) |
| month | integer | Month filter |
| year | integer | Year filter |

**Example call:**

```json
call getSessions {
  "format": "csv",
  "lang": "de",
  "month": 2,
  "year": 2025
}
```

## updateSession

Update vehicle, loadpoint or odometer of a charging session. Only provided fields are changed; a null odometer clears the stored value.

**Tags:** sessions

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| requestBody | object | The JSON request body. |

**Example call:**

```json
call updateSession {
  "id": 1,
  "requestBody": "..."
}
```

## clearCache

Clears all cached data. This resets all cached values from tariffs, vehicle APIs, and other components that use caching.

**Tags:** system

## getLogAreas

Returns a list of all log areas (e.g. `lp-1`, `site`, `db`).

**Tags:** system

## getSystemLogs

Returns the latest log lines.

**Tags:** system

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| areas | array | Comma-separated list of log areas |
| count | integer | Number of log lines to return |
| format | string | File type |
| level | string | Log level |

**Example call:**

```json
call getSystemLogs {
  "areas": [
    "lp-1",
    "site",
    "db"
  ],
  "count": 123,
  "format": "txt",
  "level": "DEBUG"
}
```

## setTelemetryStatus

Enable or disable telemetry. Note: Telemetry requires sponsorship.

**Tags:** system

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| enable | string | Charging mode. |

**Example call:**

```json
call setTelemetryStatus {
  "enable": "true"
}
```

## shutdownSystem

Shut down instance. There is no reboot command. We expect the underlying system (docker, systemd, etc.) to restart the evcc instance once it's terminated.

**Tags:** system

## getTariffInfo

Returns the prices or emission values for the upcoming hours

**Tags:** tariffs

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| type | string | Tariff type |

**Example call:**

```json
call getTariffInfo {
  "type": "grid"
}
```

## deleteVehicleMode

Resets the vehicle charge mode to keep the last selected mode.

**Tags:** vehicles

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| name | string | Vehicle name |

**Example call:**

```json
call deleteVehicleMode {
  "name": "vehicle_1"
}
```

## deleteVehicleSocPlan

Delete the charging plan

**Tags:** vehicles

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| name | string | Vehicle name |

**Example call:**

```json
call deleteVehicleSocPlan {
  "name": "vehicle_1"
}
```

## setVehicleMinSoc

Vehicle will be fast-charged until this SoC is reached.

**Tags:** vehicles

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| name | string | Vehicle name |
| soc | number | SOC in % |

**Example call:**

```json
call setVehicleMinSoc {
  "name": "vehicle_1",
  "soc": 60
}
```

## setVehicleMode

Sets the charge mode applied when this vehicle becomes active on a loadpoint.

**Tags:** vehicles

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| mode | string | Charging mode. |
| name | string | Vehicle name |

**Example call:**

```json
call setVehicleMode {
  "mode": "off",
  "name": "vehicle_1"
}
```

## setVehiclePlanStrategy

Updates the charging plan strategy for the vehicle.

**Tags:** vehicles

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| name | string | Vehicle name |
| requestBody | object | The JSON request body. |

**Example call:**

```json
call setVehiclePlanStrategy {
  "name": "vehicle_1",
  "requestBody": "..."
}
```

## setVehicleSocLimit

Charging will stop when this SoC is reached.

**Tags:** vehicles

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| name | string | Vehicle name |
| soc | number | SOC in % |

**Example call:**

```json
call setVehicleSocLimit {
  "name": "vehicle_1",
  "soc": 60
}
```

## setVehicleSocPlan

Create charging plan with fixed time and SoC target.

**Tags:** vehicles

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| name | string | Vehicle name |
| soc | number | SOC in % |
| timestamp | string | Timestamp in RFC3339 format |

**Example call:**

```json
call setVehicleSocPlan {
  "name": "vehicle_1",
  "soc": 60,
  "timestamp": "example"
}
```

## updateVehicleRepeatingPlans

Updates the repeating charging plan.

**Tags:** vehicles

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| name | string | Vehicle name |
| requestBody | array | The JSON request body. |

**Example call:**

```json
call updateVehicleRepeatingPlans {
  "name": "vehicle_1",
  "requestBody": "..."
}
```

