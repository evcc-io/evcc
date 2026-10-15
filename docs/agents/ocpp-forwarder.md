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

## Read-only mirrors

A rule can list further upstreams in `Mirrors` (`forwarder_mirror.go`). They are meant for reporting or monitoring, for example a third party that must see the charger's data while another backoffice does the billing. Unlike the main upstream, a mirror only watches:

- Every Call of the charger is copied to it. Nothing is relayed, so its replies never reach the charger and evcc's handler is never bypassed because of it.
- Calls from the mirror's upstream are answered with a `SecurityError` CallError. Results and errors from it are discarded.
- On (re)connect it first receives the cached `BootNotification` and the last `StatusNotification` per connector, with fresh message ids.
- Frames wait in a bounded backlog (oldest dropped first) while the mirror is offline and are delivered in order after it reconnects.
- Mirrors are dialled with backoff for as long as the charger is connected, are keyed by URL and station id within their rule, and are redialled when their connection parameters change.
- A mirror that reaches the same upstream station as the main connection (an omitted station id means the charger's own) or as another mirror is not started, since a second connection under one identity would displace the first. The rule is resolved and the sessions are installed under the locks that guard the charger's connection state, so reconciliations cannot overwrite each other and a charger that just disconnected gets none.
- Queued `BootNotification` and `StatusNotification` frames that the replay snapshot already covers are dropped, so a reconnect does not repeat or reorder state.
- Per mirror and hour, what was written is counted (messages, meter values, reported energy, disconnects) and persisted, together with the time of the last message. Statistics of mirrors that no rule lists anymore are removed. If the stored statistics cannot be read they are left untouched instead of being overwritten, and a failed save is retried.

Note that the main upstream is not read-only in this sense even with the `ReadOnly` flag: that flag only blocks its commands, it still answers `Authorize`, `StartTransaction`, `StopTransaction` and `DataTransfer` for the charger.

## Connection lifecycle

Frames that arrive from a charger before its sidecar finishes dialling are buffered (`pendingMsgs`) and flushed in order once the sidecar connects, so early messages such as `BootNotification` still reach upstream. If the dial fails or upstream drops mid-session, any buffered or in-flight relay Calls are answered to the charger with a `CallError` so it is not left hanging, and the failure is surfaced to the UI via `forwarderErrors`.

When the upstream connection fails while the charger stays connected, the sidecar is re-dialled automatically with exponential backoff (`runUpstreamSidecar`, 5s up to 5min; a successful session resets the backoff). Between attempts the relay actions fall back to evcc's local handler, so charging continues but upstream misses those transactions. The reconnect loop ends when the charger disconnects, the rule is removed, or its connection parameters change (`ApplyForwarderRules` dials its own sidecar in that case).

The last `BootNotification` of each connected charger is cached (`lastBoot`). When a sidecar connects mid-session (after an upstream reconnect or a rule added at runtime) and the pending buffer does not already carry a boot frame, the cached frame is replayed to upstream with a fresh message id, since many backends expect a boot before accepting transactions. Upstream's reply to the replay is discarded; evcc answered the charger's original boot long ago.

Rules can be changed at runtime through `ApplyForwarderRules`. Sidecars for removed rules are closed, rules with changed connection parameters are re-dialled, and rules for chargers that are not connected are test-dialled to surface unreachable hosts immediately.
