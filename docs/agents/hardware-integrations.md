# Hardware Integrations: Chargers, Meters, Vehicles

## Integration Pattern

All device types use a registry-based factory pattern:

```go
// Self-registration in init()
func init() {
    registry.AddCtx("typename", NewFromConfig)
}

func NewFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
    // Parse config, create client, return implementation
}
```

Optional interfaces are added via the decorator pattern:
```go
//go:generate decorate -f decorateXxx -b *Xxx -t "api.PhaseSwitcher,Phases1p3p,func(int) error"
```

## Charger Implementations

### By Protocol

| Protocol | Examples |
|----------|---------|
| HTTP/REST | Easee (REST+SignalR), Wallbox, go-e, OpenWB, Shelly |
| Modbus RTU/TCP | KEBA, Wallbe, CFOS, Bender, Delta, Mennekes |
| OCPP 1.6 | Generic charge point server |
| EEBus/ISO 15118 | EEBus SPINE protocol |
| UDP/Custom | KEBA UDP, OpenEVSE, Wattpilot, NRGKick |
| MQTT | OpenWB, Tasmota, Shelly |
| Smart Socket | Shelly, Tapo, TP-Link, FritzDECT |

### Required Charger Interface

```go
type Charger interface {
    ChargeState
    Enabled() (bool, error)
    Enable(enable bool) error
    CurrentController
}
```

Where `ChargeState` provides `Status() (ChargeStatus, error)` (A/B/C) and
`CurrentController` provides `MaxCurrent(current int64) error`.

### Optional Charger Interfaces

- `ChargerEx` — `MaxCurrentMillis(float64)` for milliamp precision
- `PhaseSwitcher` — `Phases1p3p(int)` to switch 1p/3p
- `Meter` / `MeterEnergy` — built-in power/energy measurement
- `PhaseCurrents` / `PhaseVoltages` — per-phase readings
- `ChargeRater` — `ChargedEnergy()` for session energy
- `ChargeTimer` — `ChargeDuration()` for session time
- `Identifier` — `Identify()` for RFID/vehicle identification

### Key Implementations

- **Easee** — REST + async SignalR; see `docs/agents/easee-architecture.md` for full detail
- **OCPP** (`charger/ocpp.go`) — Full 1.6 with charge point management
- **go-e** — Dual API (v1 local HTTP, v2 cloud); phase switching on v2
- **EEBus** — Complex SPINE protocol with USE cases (CEM, EV, EVCC)
- **Generic configurable** (`charger/charger.go`) — plugin-driven via YAML template

### Adding a New Charger

1. Create `charger/xxx.go` (or YAML template in `templates/definition/charger/`)
2. Implement `NewXxxFromConfig()` returning `api.Charger`
3. Required: `Status()`, `Enabled()`, `Enable()`, `MaxCurrent()`
4. Register: `registry.AddCtx("xxx", factory)` in `init()`
5. Optional: add decorator for PhaseSwitcher, Meter, etc.
6. Add template: `templates/definition/charger/xxx.yaml` for UI metadata

## Meter Implementations

### By Category

| Category | Examples |
|----------|---------|
| Modbus/SunSpec | SDM630, SMA, Fronius, Victron |
| HTTP/REST | Homewizard, Shelly Gen3, E3DC |
| Smart Home | HomeAssistant entities, Homematic |
| Battery/Storage | Tesla Powerwall, LG ESS, Zendure |

### Required Meter Interface

```go
type Meter interface {
    CurrentPower() (float64, error)  // watts
}
```

### Optional: `MeterEnergy`, `PhaseCurrents/Voltages/Powers`, `Battery`, `BatteryCapacity`

### Key Implementations
- **mbmd** (`meter/mbmd.go`) — RS485 device library with auto-detection
- **SunSpec** (`plugin/sunspec.go`) — Modbus model-based point queries
- **Generic** — plugin-driven (HTTP, Modbus, MQTT sources)

## Vehicle Integrations

| Manufacturer | API Type |
|-------------|----------|
| Tesla | Fleet API + vehicle-command proxy |
| VW Group | WeConnect (VW, Audi, Skoda, Seat, Cupra) |
| Hyundai/Kia | BlueLink (regional variants) |
| BMW/Mini | ConnectedDrive v2 |
| Mercedes | Official API |
| Renault/Nissan | Renault API + Carwings |
| Ford | FordConnect (US/EU) |
| Porsche | Porsche Connect |
| PSA Group | Peugeot, Citroen, DS, Opel |
| Generic | OVMS, Tronity |

### Required Vehicle Interface

```go
type Vehicle interface {
    Battery          // Soc() (float64, error)
    BatteryCapacity  // Capacity() float64
    IconDescriber    // Icon() string
    FeatureDescriber // Features() []Feature
    PhaseDescriber   // Phases() int
    TitleDescriber   // GetTitle() string
    SetTitle(string)
    Identifiers() []string
    OnIdentified() ActionConfig
}
```

### Optional: `SocLimiter`, `ChargeState`, `VehicleRange`, `VehicleOdometer`, `VehicleClimater`, `VehicleFinishTimer`, `VehiclePosition`, `CurrentLimiter`, `CurrentController`, `ChargeController`, `Resurrector`

### Polling Strategy

Configurable: always / while charging / while connected. Interval-based caching
to avoid excessive cloud API calls. OAuth2 token handling built into each provider.

## Auto-Detection (`cmd/detect/`)

Task-based parallel IP scanning:
`ping` -> `tcp_http` -> `tcp_modbus` -> `sunspec` -> device-specific probes

Detects: OpenWB, SMA, KEBA, E3DC, Sonnen, Tesla Powerwall, Wallbe, Fronius,
Tasmota, Shelly, Phoenix, and many more.
