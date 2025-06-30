# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

evcc is an extensible EV Charge Controller and home energy management system written in Go with a Vue.js frontend. It manages electric vehicle charging, integrates with solar systems, and provides local energy management without cloud dependencies.

## Core Development Commands

### Build & Development

- `make` - Build the full application (UI + Go binary)
- `make build` - Build Go binary only
- `make ui` - Build UI assets only
- `go run ./...` - Run without building binary
- `./evcc` - Run the built binary
- `./evcc --demo` - Run with demo configuration

### Dependencies

- `make install` - Install Go tools and dependencies
- `make install-ui` - Install Node.js dependencies (`npm ci`)

### Testing & Quality

- `make test` - Run Go tests
- `make test-ui` - Run frontend tests (`npm test`)
- `make lint` - Run Go linting (golangci-lint)
- `make lint-ui` - Run frontend linting (Prettier, ESLint, TypeScript)

### Frontend Development

- `npm run dev` - Start Vue dev server (http://127.0.0.1:7071)
- `npm run storybook` - Run Storybook (http://127.0.0.1:6006)
- `npm run playwright` - Run integration tests
- `npm run simulator` - Run device simulator (http://localhost:7072)

### Device Templates

- `evcc --template-type charger --template new-charger-template.yaml` - Test device templates
- `make docs` - Generate template documentation

## Architecture

### Core Components

- **main.go** - Entry point, embeds web assets and i18n files
- **cmd/** - CLI commands and application setup
- **core/** - Core business logic:
  - **loadpoint.go** - EV charging point management
  - **site.go** - Site-wide energy management
  - **planner/** - Smart charging planning
  - **coordinator/** - Multi-loadpoint coordination
- **api/** - API definitions and types
- **server/** - HTTP server, WebSocket, MQTT, and database
- **charger/**, **meter/**, **vehicle/** - Device integrations
- **tariff/** - Tariff integrations
- **plugin/** - Plugin system for device and tariff communication
- **assets/** - Vue.js frontend application

### Frontend Structure

- **assets/js/** - TypeScript/Vue.js application
- **assets/views/** - Vue components and pages
- **i18n/** - Internationalization files
- **dist/** - Built frontend assets (generated)

### Configuration

- Uses YAML configuration files
- Templates in `templates/definition/` for device and tariff configurations
- Database: SQLite (default: `evcc.db`)

## Key Development Patterns

### Device Integration

- Device types: chargers, meters, vehicles (and tariffs)
- Plugin system supports: Modbus, HTTP, MQTT, JavaScript, Go
- Templates define device capabilities and configuration
- Use `_blueprint.go` as starting point for new Go implementations

### Testing

- Go tests use standard `testing` package
- Frontend tests use Vitest
- Integration tests use Playwright
- Simulator available for testing without real devices

### Internationalization

- Translations managed via Weblate
- Update both `i18n/de.json` and `i18n/en.json` for new strings
- Use `$t()` function in Vue components

## Important File Locations

- Configuration: `evcc.yaml` (or specified with `--config`)
- Database: `evcc.db` (or specified with `--database`)
- Device templates: `templates/definition/`
- Web assets: `assets/` (source), `dist/` (built)
- Generated code: Files ending in `_enumer.go`, `*_decorators.go`

## Common Development Workflows

### Adding New Device Support

1. Create template in `templates/definition/[type]/`
2. If Go code needed, implement in respective package
3. Test with `evcc --template-type [type] --template [file]`
4. Run `make docs` to update documentation

### Frontend Development

1. Start backend: `make build && ./evcc`
2. Start frontend dev server: `npm run dev`
3. Access development UI at http://127.0.0.1:7071

### Integration Testing

1. Build application: `make ui build`
2. Run tests: `npm run playwright`
3. Use simulator for device testing: `npm run simulator`
