The charging-status messages are a wire-compatible subset of the definitions
used by [mbapi2020](https://github.com/ReneNulschDE/mbapi2020). Only
`PushMessage.vehicle_status_updates` (24), its vehicle map (2),
`VehicleStatusUpdate.full_update` (2), `chargingstatus` (50), and the status
attribute's value (1) and metadata (2) are retained. Metadata status 0 means
valid. Command-status notifications (17) are acknowledged (19) without
interpreting or executing commands; vehicle-status updates are acknowledged
with field 28. Other fields are discarded during decoding.

Regenerate with `go generate -tags mercedes_proto ./vehicle/mercedes` using
`protoc` 36.2 and `protoc-gen-go` 1.36.12 on PATH. Regeneration is opt-in so
normal asset generation does not require protobuf compiler tools. The schema is deliberately separate from the existing
REST protobuf definitions, which do not need regeneration.

The initial WebSocket snapshot is read on demand with a 30-second timeout.
The connection is then closed. A shared account cache limits connection
attempts, including failures, to once per minute even when vehicle detection
resets the normal provider caches. An absent or invalid attribute and unknown
status values produce an error, never a fabricated disconnected state.
