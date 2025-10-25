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
  "enable": "example"
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
  "soc": 123.45
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
  "soc": 123.45
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
  "batteryMode": "example"
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
  "soc": 123.45
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
  "power": 123.45
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

## healthCheck

Returns 200 if the evcc loop runs as expected.

**Tags:** general

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
  "id": 123,
  "name": "example"
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
  "id": 123
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
  "id": 123
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
  "id": 123
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
  "id": 123
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
  "energy": 123.45,
  "id": 123,
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
  "hourMinuteTime": "example",
  "id": 123,
  "soc": 123.45,
  "timezone": "example",
  "weekdays": "..."
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
  "id": 123,
  "soc": 123.45,
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
  "id": 123
}
```

## setLoadpointBatteryBoost

Enable or disable battery boost.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| enable | string | Charging mode. |
| id | integer | Loadpoint index starting at 1 |

**Example call:**

```json
call setLoadpointBatteryBoost {
  "enable": "example",
  "id": 123
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
  "delay": 123,
  "id": 123
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
  "id": 123,
  "threshold": 123.45
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
  "delay": 123,
  "id": 123
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
  "id": 123,
  "threshold": 123.45
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
  "energy": 123.45,
  "id": 123
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
  "energy": 123.45,
  "id": 123,
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
  "current": 123.45,
  "id": 123
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
  "current": 123.45,
  "id": 123
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
  "id": 123,
  "mode": "example"
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
  "id": 123,
  "phases": "example"
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
  "id": 123,
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
  "id": 123
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
  "id": 123
}
```

## setLoadpointSocLimit

Updates the SoC limit of the loadpoint. Requires a connected vehicle with known SoC. Limit is maintained across charging sessions.

**Tags:** loadpoints

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| soc | number | SOC in % |

**Example call:**

```json
call setLoadpointSocLimit {
  "id": 123,
  "soc": 123.45
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
  "id": 123
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
  "id": 123
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
  "format": "example",
  "lang": "example",
  "month": 123,
  "year": 123
}
```

## updateSession

Update vehicle of charging session.

**Tags:** sessions

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| id | integer | Loadpoint index starting at 1 |
| requestBody | object | The JSON request body. |

**Example call:**

```json
call updateSession {
  "id": 123,
  "requestBody": "..."
}
```

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
  "areas": "...",
  "count": 123,
  "format": "example",
  "level": "example"
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
  "enable": "example"
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
  "type": "example"
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
  "name": "example"
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
  "name": "example",
  "soc": 123.45
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
  "name": "example",
  "soc": 123.45
}
```

## setVehicleSocPlan

Create charging plan with fixed time and SoC target.

**Tags:** vehicles

**Arguments:**

| Name | Type | Description |
|------|------|-------------|
| name | string | Vehicle name |
| precondition | integer | Late charging duration in seconds. |
| soc | number | SOC in % |
| timestamp | string | Timestamp in RFC3339 format |

**Example call:**

```json
call setVehicleSocPlan {
  "name": "example",
  "precondition": 123,
  "soc": 123.45,
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
  "name": "example",
  "requestBody": "..."
}
```

