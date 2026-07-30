# Easee Charger Architecture

The Easee integration communicates with the Easee cloud over two distinct channels:

1. **REST API** (`https://api.easee.com/api`) — synchronous control commands and configuration.
2. **SignalR WebSocket** (`https://streams.easee.com/hubs/chargers`) — asynchronous real-time state updates and command confirmations.

Commands are sent via REST; their acknowledgement and state changes are delivered asynchronously via SignalR.

## Authentication

### Flow

Authentication uses username/password credentials against:
```
POST https://api.easee.com/api/accounts/login
```

Returns a `Token` struct with `accessToken` (short-lived JWT), `refreshToken`, `expiresIn`, and `tokenType`.

### Token Lifecycle

Wrapped in `oauth2.TokenSource` via `oauth.RefreshTokenSource`. Near expiry, automatically calls:
```
POST https://api.easee.com/api/accounts/refresh_token
```
Falls back to full re-login if refresh fails.

### Token Caching

`TokenSource` is shared per user via `cache.New[oauth2.TokenSource]()`. Multiple `Easee` instances with the same user email share a single token source, preventing redundant re-authentication.

## Initialization Sequence

`NewEasee` performs these steps in order:

1. **Charger Discovery** — If no serial provided, queries `GET /api/chargers` and expects exactly one charger.
2. **Site and Circuit Discovery** — `GET /api/chargers/{chargerID}/site`. Searches for a single-charger circuit for circuit-level phase control.
3. **SignalR Connection** — Creates client with `WithMaxElapsedTime(0)` to retry forever (default 15-min cap would silently stop updates).
4. **Subscription** — On every `ClientConnected`, sends `SubscribeWithCurrentState(chargerID, true)` to replay full current state before switching to push-on-change.
5. **Startup Gate** — Blocks until `CHARGER_OP_MODE` is received (one-shot `sync.OnceFunc`).
6. **Optional State Wait** — Waits up to 3s for `SESSION_ENERGY`, `LIFETIME_ENERGY`, `TOTAL_POWER`. WARN if missing but initialization succeeds.

## SignalR Back-Channel

### Why SignalR is Required

1. **Commands are fire-and-forget at HTTP level.** HTTP response only confirms cloud received the request. Success/failure arrives via SignalR `CommandResponse`.
2. **State is event-driven, not pollable.** No REST endpoint streams charger state.
3. **Ticks correlation only works with a live connection.** If SignalR drops mid-command, the waiter times out.

### Server -> Client Methods

#### `ProductUpdate(json.RawMessage)`
Primary state channel. Carries a single `Observation` with `ID` (ObservationID), `Value`, `DataType`, and `Timestamp`.

- **Timestamp deduplication**: older timestamps for the same ID are silently dropped.
- **Non-blocking fan-out**: observation sent on `obsC` via non-blocking select.

#### `CommandResponse(json.RawMessage)`
Async acknowledgement for REST commands. Contains `Ticks` (correlation key), `WasAccepted`, `ResultCode`, and `ID` (ObservationID).

Routes through three maps in order:
1. `pendingTicks[res.Ticks]` — primary correlation for async (HTTP 202) commands
2. `pendingByID[ObservationID(res.ID)]` — fallback when Ticks mismatch
3. `expectedOrphans[ObservationID(res.ID)]` — counter for sync (HTTP 200) endpoints that still produce a CommandResponse

Unmatched responses are logged as WARN (rogue response from external system).

#### `ChargerUpdate` / `SubscribeToMyProduct`
Logged at TRACE, not processed further.

## Command Flow and Async Correlation

### REST Command Endpoints

```
POST /api/chargers/{chargerID}/commands/{action}     (start/stop/pause/resume)
POST /api/chargers/{chargerID}/settings              (enable, DCC, PhaseMode, SmartCharging)
POST /api/sites/{siteID}/circuits/{circuitID}/settings  (dynamic circuit currents)
```

### Response Handling

| HTTP Status | Meaning | Behavior |
|-------------|---------|----------|
| `200` | Synchronous / already applied | Returns immediately |
| `202` | Asynchronous, Ticks provided | Waits for matching CommandResponse |
| other | Error | Returns error |

### Ticks Correlation

On 202, the body contains `RestCommandResponse` with a `Ticks` field (.NET DateTime.Ticks). If `Ticks == 0`, the command was a no-op.

Each in-flight command creates a **buffered channel** (capacity 1), registered in both `pendingTicks` and `pendingByID`, cleaned up via `defer`.

### The Sync/Async Mismatch

Some endpoints return HTTP `200` but still fire a `CommandResponse` via SignalR. The observed case is circuit settings (`POST /api/sites/{siteID}/circuits/{circuitID}/settings`) which returns `200` but generates `CommandResponse` with `ID=22` (`CIRCUIT_MAX_CURRENT_P1`).

Handled via the **expected-orphan counter**:
```go
expectedOrphans map[easee.ObservationID]int  // protected by cmdMu
```

Before a POST to a known 200-returning endpoint, increment the counter. When `CommandResponse` arrives with no pending match, decrement and silently consume. Counter at 0 means genuinely rogue.

## State Management

### Internal State Fields (all protected by `sync.RWMutex`)

| Field | Observation | Notes |
|-------|------------|-------|
| `opMode` | `CHARGER_OP_MODE` (109) | Central state machine |
| `chargerEnabled` | `IS_ENABLED` (31) | Hardware enable state |
| `smartCharging` | `SMART_CHARGING` (102) | LED color mode |
| `currentPower` | `TOTAL_POWER` (120) | Watts (API sends kW, multiplied by 1000) |
| `sessionEnergy` | `SESSION_ENERGY` (121) | kWh, special zero-handling |
| `totalEnergy` | `LIFETIME_ENERGY` (124) | kWh, updated ~hourly |
| `currentL1/L2/L3` | `IN_CURRENT_T3/T4/T5` (183/184/185) | Phase currents in A |
| `phaseMode` | `PHASE_MODE` (38) | 1=single, 2=auto, 3=locked 3-phase |
| `dynamicCircuitCurrent[3]` | `DYNAMIC_CIRCUIT_CURRENT_P1/P2/P3` (111/112/113) | Per-phase circuit limit |
| `maxChargerCurrent` | `MAX_CHARGER_CURRENT` (47) | Hardware max (non-volatile) |
| `dynamicChargerCurrent` | `DYNAMIC_CHARGER_CURRENT` (48) | Volatile current limit |
| `reasonForNoCurrent` | `REASON_FOR_NO_CURRENT` (96) | Debug enum |
| `pilotMode` | `PILOT_MODE` (100) | CP signal state A-F |
| `rfid` | `USER_IDTOKEN` (128) | Last scanned RFID token |

### Session Energy Zero-value Protection

`sessionEnergy` is never set to `0` from a `ProductUpdate` — the API sends spurious zeros erratically. Session reset is driven by op-mode transition: when `CHARGER_OP_MODE` transitions from disconnected to awaiting-start, `sessionEnergy` resets to `0` with a fresh timestamp.

## Charger Operation Modes

```
0 = Offline               — no cloud connection
1 = Disconnected          — no car plugged in
2 = AwaitingStart         — car plugged, waiting for authorization/start
3 = Charging              — actively charging
4 = Completed             — car full or finished, cable still plugged
5 = Error                 — fault condition
6 = ReadyToCharge         — ready, current available
7 = AwaitingAuthentication — RFID auth required
8 = Deauthenticating      — finishing authentication teardown
```

### Mapping to evcc Status

| opMode | evcc Status |
|--------|------------|
| 1 (Disconnected) | A |
| 2, 4, 6, 7, 8 | B |
| 3 (Charging) | C |
| 0, 5 and others | error |

## Enable/Disable Flow

### Enable = true

1. If `chargerEnabled == false`: POST settings `{ enabled: true }` and wait.
2. If `opMode == Disconnected`: return (no cable).
3. If `opMode == AwaitingAuthentication && authorize`: action = `start_charging`.
4. Otherwise: action = `resume_charging`.
5. POST `/commands/{action}` and wait.
6. Wait for `opMode` to reach enabled state.
7. Wait for `dynamicChargerCurrent` to reach `32` (Easee sets this on resume).
8. Call `MaxCurrent(c.current)` to restore previous setpoint.

### Enable = false

1. If disconnected or (awaiting auth && !authorize): return.
2. POST `/commands/pause_charging` and wait.
3. Wait for `opMode` to reach disabled state.
4. Wait for `dynamicChargerCurrent` to reach `0`.

### State Waiting Pattern

Both `waitForChargerEnabledState` and `waitForDynamicChargerCurrent` use:
1. Short-circuit check: if already in target state, return immediately.
2. Open a timer.
3. Loop on `obsC` channel.
4. On timer expiry: **one final check** before returning `api.ErrTimeout`.

The final check handles the race where the state update arrived between the last channel read and the timer fire.

## Phase Control

### Circuit-Level (preferred, when circuit is known)

Phase switching by zeroing dynamic circuit current on unused phases:
```
POST /api/sites/{siteID}/circuits/{circuitID}/settings
```

For 1-phase: set P2=0, P3=0. For 3-phase: restore all three.

This POST returns HTTP `200` but still fires a `CommandResponse` with `ID=22` (expected orphan).

### Charger-Level (fallback)

Uses `PhaseMode` setting: `1` for single-phase, `2` (auto) for 3-phase.
After changing PhaseMode, `Enable(false)` is called — the loadpoint then re-enables, because PhaseMode changes only take effect after a charging cycle restart.

## Authorization Mode (`authorize`)

When `authorize: true`, evcc sends `start_charging` to authorize sessions when the charger enters `ModeAwaitingAuthentication`. This enables fully unattended operation but is incompatible with RFID-based vehicle identification.

When `authorize: false`, evcc does nothing in mode 7 — the charger waits for external authorization (RFID card or app).

Setting `authorize: true` also prevents the charger from auto-starting at 32A on plug-in, giving evcc full control from the first amp.

## Concurrency Model

### Mutexes

| Mutex | Type | Protects |
|-------|------|----------|
| `c.mux` | `sync.RWMutex` | All charger state fields |
| `dispatcher.mu` | `sync.Mutex` | `pendingTicks`, `pendingByID`, `expectedOrphans` maps (inside `CommandDispatcher`) |

Command dispatch was extracted into `charger/easee/dispatcher.go` (`CommandDispatcher` struct). The two mutexes are intentionally separate to prevent the SignalR receive loop from blocking on command dispatch operations.

### Observation Channel

`obsC chan Observation` is unbuffered. `ProductUpdate` sends via non-blocking select — if no waiter is listening, the notification is dropped. The authoritative state is always in the struct fields; the channel is only a notification mechanism.

**Design constraint**: any waiter on `obsC` must include a final state check after timer expiry before returning `api.ErrTimeout`.

## Known Design Concerns

1. **SESSION_ENERGY zero-value protection** — defensive measure based on field observations; root cause unverified.
2. **LIFETIME_ENERGY** — inaccurate by design, API pushes updates ~hourly.
3. **current vs dynamicChargerCurrent drift** — evcc's desired setpoint and charger's confirmed value can drift around pause/resume cycles. Resynced via `MaxCurrent(c.current)` after resume.
4. **Multi-charger circuits** — only circuit-level phase control when charger is alone on its circuit. Multi-charger circuits fall back to less precise charger-level control.
5. **Stale CommandResponses after reconnect** — if SignalR drops mid-command, the response may arrive after reconnect with no pending entry, triggering a false-positive rogue WARN. Acceptable trade-off.

## API Endpoints Summary

| Method | Endpoint | Used For |
|--------|----------|----------|
| `POST` | `/accounts/login` | Initial authentication |
| `POST` | `/accounts/refresh_token` | Token refresh |
| `GET` | `/chargers` | Auto-discover charger ID |
| `GET` | `/chargers/{id}/site` | Discover site and circuit |
| `POST` | `/chargers/{id}/settings` | Enable/disable, DCC, PhaseMode, SmartCharging |
| `POST` | `/chargers/{id}/commands/{action}` | start/stop/pause/resume charging |
| `GET` | `/sites/{siteId}/circuits/{circuitId}/settings` | Read max circuit currents |
| `POST` | `/sites/{siteId}/circuits/{circuitId}/settings` | Set dynamic circuit currents (phase switching) |

### SignalR Hub

| Endpoint | `https://streams.easee.com/hubs/chargers` |
|----------|------------------------------------------|
| Client -> Server | `SubscribeWithCurrentState(chargerID, true)` |
| Server -> Client | `ProductUpdate`, `ChargerUpdate`, `SubscribeToMyProduct`, `CommandResponse` |

## Configuration

| Parameter | Required | Default | Notes |
|-----------|----------|---------|-------|
| `user` | yes | | Easee account email |
| `password` | yes | | Easee account password |
| `charger` | no | | Charger serial; auto-detected if exactly one on account |
| `timeout` | no | `20s` | HTTP timeout for all API calls and command waits |
| `authorize` | no | `false` | If true, evcc sends `start_charging` to authorize sessions |

Supported products: Easee Home, Easee Charge, Easee Charge Lite, Easee Charge Core.
Declared capabilities: `1p3p` (phase switching), `rfid` (RFID identification).
Requires evcc sponsorship.
