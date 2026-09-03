# Mercedes protobuf definitions

These `*.pb.go` files are generated protobuf bindings for the Mercedes-Benz
mobile SDK ("RIS") backend. Mercedes does not publish the `.proto` sources.

## Provenance

The definitions track the ones used by the actively maintained Home Assistant
integration [ReneNulschDE/mbapi2020](https://github.com/ReneNulschDE/mbapi2020),
which in turn mirror the iOS `MBSDK-CarKit-iOS/Proto/*.proto` sources. The
bindings here were regenerated from the `FileDescriptorProto`s embedded in
mbapi2020's compiled `*_pb2.py` files.

## Layout

- `package protos` for every file; `protos.proto` lives in the `protos/`
  subdirectory (import path `.../vehicle/mercedes/pb/protos`), everything else
  in this directory (import path `.../vehicle/mercedes/pb`). Only `api.go`,
  `vsu.go` and `websocket.go` import these packages.
- The `gogo.proto` custom options used by some of the upstream `.proto` files
  are stripped during generation (they do not affect the wire format), so this
  package does not depend on `github.com/gogo/protobuf`.

## Regeneration

Requires `protoc-gen-go` on `PATH`. Extract the descriptors from a checkout of
mbapi2020's `custom_components/mbapi2020/proto/*_pb2.py`, assemble a
`FileDescriptorSet`, map each file's `go_package` to the layout above (drop
gogo, keep the well-known types' canonical import paths), and drive
`protoc-gen-go` with a `CodeGeneratorRequest` (`paths=source_relative`). See the
commit that introduced the `VehicleStatusUpdate` (VSU) messages for the exact
tooling used.

## Key VSU fields (used by vsu.go)

| StatusResponse            | VSU field                 | Type / field no.              |
| ------------------------- | ------------------------- | ----------------------------- |
| Odometer                  | `odo`                     | Int64DistanceAttribute, 147   |
| Soc                       | `soc`                     | Int64RatioAttribute, 196      |
| Range                     | `rangeelectric`           | Int64DistanceAttribute, 183   |
| EndOfChargeTime           | `endofchargetime`         | Int64ClockHourAttribute, 96   |
| ChargingStatus            | `chargingstatus`          | ChargingstatusEnumAttribute,50|
| SelectedChargeProgram     | `selected_charge_program` | EnumAttribute, 190            |
| SocLimit                  | `max_soc`/`charge_programs`| Int64RatioAttribute 138 / 27 |
| Preconditioning           | `precond_active`          | BoolAttribute, 167            |
| Position                  | `position_lat`/`_long`    | DoubleAttribute, 165/166      |

The ack for `vehicle_status_updates` (PushMessage field 24) is
`ClientMessage.acknowledge_vehicle_status_updates` (field 28).
