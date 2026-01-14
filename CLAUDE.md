# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is evcc?

evcc is an extensible EV Charge Controller and home energy management system written in Go with a Vue.js frontend. It manages solar surplus charging, supports 100+ charger brands, and integrates with meters, vehicles, and dynamic tariffs.

## Common Commands

### Prerequisites (one-time)
```sh
make install-ui   # npm ci
make install      # go install tool
```

### Build & Run
```sh
make              # builds UI + Go binary (./evcc)
go run ./...      # run without building binary
./evcc -c cmd/demo.yaml  # run with demo config
```

### Testing
```sh
make test         # Go tests: CGO_ENABLED=0 go test -tags=release ./...
make test-ui      # npm test (vitest)
go test ./core/... -run TestLoadpoint  # run specific Go tests
npm run playwright  # E2E tests (requires: make ui build)
```

### Linting
```sh
make lint         # golangci-lint + go tool modernize
make lint-ui      # npm run lint (prettier + eslint + tsc + i18n check)
```

### UI Development
```sh
npm run dev       # Vite dev server at :7071 (proxies to :7070)
npm run storybook # Component development at :6006
npm run simulator # Device simulator at :7072
```

### Template Documentation
```sh
make docs         # generates template docs to /templates/docs
```

## Architecture Overview

### Core Components (core/)
- **Site**: Top-level coordinator aggregating meters (grid, PV, battery) and managing loadpoints
- **Loadpoint**: State machine per charging point handling mode logic (off/now/minpv/pv)
- **Coordinator**: Tracks vehicle-to-loadpoint assignments to prevent double-counting
- **Planner**: Smart-cost charging scheduler using tariff data

### Interface-Based Design (api/api.go)
Components implement small, composable interfaces rather than fat interfaces:
- `Meter`, `MeterEnergy`, `PhaseCurrents`, `PhaseVoltages` for metering
- `Charger`, `ChargerEx` (milliamp precision), `PhaseSwitcher` for charging
- `Battery`, `BatteryController` for storage systems

Devices compose only the interfaces they support, enabling flexible capability detection.

### Template System (templates/definition/)
Device integrations are defined as YAML templates with:
- Parameter metadata (description, validation, defaults)
- Go text/template rendering for config generation
- Preset inheritance (`preset: vehicle-common`)
- Embedded at build time via `//go:embed`

Templates can use the plugin system or reference dedicated Go implementations.

### Plugin System (plugin/)
Extensibility layer for device communication:
- Protocol plugins: `modbus`, `mqtt`, `http`, `websocket`, `script`
- Auth plugins: `oauth2`, `basic`
- Pipeline for value transformation (regex, jq, math)
- Registered via `util/registry` factory pattern

### Wrapper Pattern (core/wrapper/)
Synthetic implementations when hardware lacks features:
- `ChargeMeter`: Calculates power from charger current setting
- `ChargeRater`: Estimates energy from current over time
- `ChargeTimer`: Tracks charging duration

### WebSocket-First Frontend (assets/js/)
- Single WebSocket at `/ws` pushes all state changes
- Keys use dot notation: `loadpoints.0.power`, `grid.power`
- Store merges updates via `setProperty()` for reactive UI
- No request-response for state - push-only architecture

### Key Directories
- **api/**: Interface definitions (not HTTP endpoints)
- **core/**: Business logic (Site, Loadpoint, Coordinator, Planner)
- **cmd/**: CLI commands, config loading, device detection
- **server/**: HTTP/WebSocket handlers, REST API
- **plugin/**: Protocol implementations and auth
- **charger/**, **meter/**, **vehicle/**: Device-specific implementations
- **tariff/**: Dynamic pricing integrations (Tibber, Awattar, Octopus)
- **hems/**: Home energy management integrations

## Development Notes

- Go 1.25+ and Node 22+ required
- Build tags: use `-tags=release` for production builds
- Demo config at `cmd/demo.yaml` for local testing
- Simulator at `tests/simulator.evcc.yaml` for testing without hardware
- Translations in `i18n/*.json` - managed via Weblate
- New translations require entries in both `de.json` and `en.json`
