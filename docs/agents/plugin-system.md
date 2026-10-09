# Plugin System

The plugin system (`plugin/`) provides protocol-level abstraction for device
communication. Plugins implement typed getter/setter interfaces and are composed
into charger, meter, or vehicle implementations via configuration.

## Plugin Types

| Plugin | Protocol | Key Config |
|--------|----------|------------|
| `http` | HTTP/REST | `uri`, `method`, `headers`, `auth`, `cache`, `timeout` |
| `mqtt` | MQTT | `topic`, `retained`, `payload` template, `timeout` |
| `modbus` | Modbus TCP/RTU | `uri`, `register`, `scale`, `baudrate`, `rtu` |
| `sunspec` | SunSpec/Modbus | Model-based point queries via device tree |
| `js` | JavaScript/WASM | Inline script evaluation |
| `go` | Go runtime | Dynamic Go code |
| `gpio` | Linux GPIO | Digital I/O for relays |

## Getter/Setter Interfaces

```go
type StringGetter func() (string, error)
type FloatGetter  func() (float64, error)
type IntGetter    func() (int64, error)
type BoolGetter   func() (bool, error)
// + corresponding Setter types
```

## Pipeline Transforms

Plugins support chained transforms: `scale`, `offset`, `lookup`, `regex`.

## Template-Based Device Configuration

Devices can be defined entirely via YAML templates using plugins:

```yaml
# templates/definition/charger/example.yaml
status:
  source: http
  uri: http://{{ .host }}/status
enable:
  source: http
  uri: http://{{ .host }}/enable
  method: POST
maxcurrent:
  source: http
  uri: http://{{ .host }}/current/{{ .maxcurrent }}
```

The generic configurable charger (`charger/charger.go`) wires these plugin
configs into the `api.Charger` interface at runtime.

Templates render once at config time. HTTP `uri` and `body` are templates
themselves, evaluated per request (`util.ReplaceFormatted`: sprig, `addDate`,
`timeRound`). Defer `now` to request time with a raw string:
`` {{ `{{ now | date "2006-01-02" }}` }} ``.

## Param Suggestions and Discovery

Template params get suggestions from services (`service:`), see the `service`
and `discovery` sections in `templates/README.md` for the contract. Handlers
register in `init()` via `service.Register` (`server/service/registry.go`),
object responses use `service.Option` (`util/service/helper.go`) and render as
`Helper/Combobox.vue`.

Local network discovery lives in `util/discovery` (mDNS, SSDP, neighbor table,
reverse DNS; no raw sockets, no port scans) and feeds `network/hosts`,
`network/report` and `network/scan` in `util/service/network*.go`. The vendor
list is generated from the IEEE registry (`make oui-update`), see
[Discovery Hints](discovery-hints.md) for the update routine.

- Scan: 3 s first pass (first request waits), 5 s second pass, repeated when
  older than a minute, results combined, hosts dropped after 30 minutes.
- mDNS: browses the types of the template hints plus all types announced in
  the network (DNS-SD enumeration), so reports show vendor types without a hint.
- A host keeps all its names (router name first, then mDNS), hostname hints
  match any of them.
- `EVCC_DISCOVERY_HOSTS` (JSON list of `discovery.Host`) replaces the scan,
  used in tests and for environments without network access. On macOS the neighbor table is empty for binaries started with
  `go run`, test with a built binary.

## Key Files

- `plugin/config.go` — plugin registry and config types
- `plugin/http.go` — HTTP plugin
- `plugin/mqtt.go` — MQTT plugin
- `plugin/modbus.go` — Modbus plugin
- `plugin/sunspec.go` — SunSpec plugin
- `charger/charger.go` — generic configurable charger using plugins
