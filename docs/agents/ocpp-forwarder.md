# OCPP Forwarder Architecture

The OCPP forwarder (`charger/ocpp/forwarder.go`) is a hybrid proxy that lets a charger talk to evcc and an upstream OCPP server at the same time. Chargers connect directly to evcc's central system on the normal port. For each charger with a matching `ForwarderRule`, a "sidecar" WebSocket connection to the upstream server is opened and kept in parallel for the lifetime of the charger connection.

The forwarder is opt-in: hooks (`chargerConnectHook`, `chargerDisconnectHook`, `chargerMessageHook` in `instance.go`) are nil unless a rule matches, so a charger without a rule behaves exactly as before.

## Forwarding modes

Two modes apply at the same time, selected per message by its action.

### Transparent relay (billing-critical)

For the actions in `actionsRelayedToUpstream` (`Authorize`, `StartTransaction`, `StopTransaction`, `DataTransfer`), upstream is the authoritative Central System:

1. Charger sends the Call to evcc.
2. The message hook forwards it to the upstream sidecar and bypasses evcc's OCPP handler.
3. Upstream's `CallResult`/`CallError` is relayed back to the charger.

evcc's handler is never invoked for these. This lets the pay backend control authorization, issue its own transaction IDs, and see consistent Start/Stop pairs.

### Sidecar observation (informational)

For all other messages (`BootNotification`, `StatusNotification`, `MeterValues`, `Heartbeat`, etc.):

1. Charger sends the Call to evcc, which processes it normally.
2. The same frame is also mirrored to the upstream sidecar.

Upstream observes the session while evcc manages the charger as usual.

## Upstream to charger (commands)

Calls (type 2) initiated by upstream are injected into the charger via `CS.Write`. The charger's `CallResult`/`CallError` is routed back to upstream. Examples: `RemoteStartTransaction`, `RemoteStopTransaction`, `GetConfiguration`, `ChangeConfiguration`, `TriggerMessage`, `SetChargingProfile`.

`ChangeConfiguration` for `MeterValueSampleInterval` is intercepted: the forwarder absorbs it as a local throttle on `MeterValues` forwarded to upstream and replies `Accepted` without touching the charger's own config. evcc still processes every `MeterValues` frame for energy management.

## Read-only mode

When a rule sets `ReadOnly`, upstream may observe but cannot control the charger. Any incoming Call from upstream is answered with a `SecurityError` and not forwarded. `ReadOnly` is applied live per message, so toggling it does not require reconnecting the sidecar.

## Connection lifecycle

Frames that arrive from a charger before its sidecar finishes dialling are buffered (`pendingMsgs`) and flushed in order once the sidecar connects, so early messages such as `BootNotification` still reach upstream. If the dial fails or upstream drops mid-session, any buffered or in-flight relay Calls are answered to the charger with a `CallError` so it is not left hanging, and the failure is surfaced to the UI via `forwarderErrors`.

When the upstream connection fails while the charger stays connected, the sidecar is re-dialled automatically with exponential backoff (`runUpstreamSidecar`, 5s up to 5min; a successful session resets the backoff). Between attempts the relay actions fall back to evcc's local handler, so charging continues but upstream misses those transactions. The reconnect loop ends when the charger disconnects, the rule is removed, or its connection parameters change (`ApplyForwarderRules` dials its own sidecar in that case).

The last `BootNotification` of each connected charger is cached (`lastBoot`). When a sidecar connects mid-session (after an upstream reconnect or a rule added at runtime) and the pending buffer does not already carry a boot frame, the cached frame is replayed to upstream with a fresh message id, since many backends expect a boot before accepting transactions. Upstream's reply to the replay is discarded; evcc answered the charger's original boot long ago.

Rules can be changed at runtime through `ApplyForwarderRules`. Sidecars for removed rules are closed, rules with changed connection parameters are re-dialled, and rules for chargers that are not connected are test-dialled to surface unreachable hosts immediately.
