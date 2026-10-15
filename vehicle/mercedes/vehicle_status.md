The vehicle-status messages are a wire-compatible subset of the definitions
used by [mbapi2020](https://github.com/ReneNulschDE/mbapi2020). Only attributes
consumed by existing evcc interfaces are retained:

- `chargingstatus` (50): `api.ChargeState`.
- `max_soc` (138): `api.SocLimiter`, read only.
- `odo` (147): `api.VehicleOdometer`, kilometers or miles converted to kilometers.
- `position_lat` (165), `position_long` (166): `api.VehiclePosition`.
- `precond_active` (167): `api.VehicleClimater`, read only.

These are carried in `PushMessage.vehicle_status_updates` (24), vehicle map
(2), and a full `VehicleStatusUpdate` (`full_update`, field 2). Attribute value
(1), metadata (2) and distance unit (3) are retained. Metadata status 0 means
valid. Command-status notifications (17) are acknowledged (19) without
interpreting or executing commands; vehicle-status updates are acknowledged
with field 28. Other fields are discarded during decoding. SoC and range keep
their existing REST source.

Regenerate with `go generate -tags mercedes_proto ./vehicle/mercedes` using
`protoc` 36.2 and `protoc-gen-go` 1.36.12 on PATH. Regeneration is opt-in so
normal asset generation does not require protobuf compiler tools. The schema
is separate from the existing REST protobuf definitions.

The initial full WebSocket snapshot is read on demand with a 30-second timeout.
The connection is then closed. A shared account cache limits connection
attempts, including failures, to once per minute even when vehicle detection
resets the normal provider caches. Partial updates cannot complete this
snapshot. Each getter validates its own attribute metadata. Missing or invalid
attributes, unknown charging states or distance units, and out-of-range
positions produce errors instead of fabricated default values. A program-specific
SoC limit (`max_soc` 0) is reported as unavailable; charge-program interpretation
is outside this subset.
