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

## Key Files

- `plugin/config.go` — plugin registry and config types
- `plugin/http.go` — HTTP plugin
- `plugin/mqtt.go` — MQTT plugin
- `plugin/modbus.go` — Modbus plugin
- `plugin/sunspec.go` — SunSpec plugin
- `charger/charger.go` — generic configurable charger using plugins
