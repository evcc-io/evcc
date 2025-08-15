# Agent Rules for evcc Project

This file provides guidance to AI coding agents when working with code in this repository.

## Project Overview

- evcc is an extensible EV Charge Controller and home energy management system written in Go with a Vue.js frontend
- The system manages electric vehicle charging, integrates with solar systems, and provides local energy management without cloud dependencies
- Architecture follows a plugin-based approach for device integrations

## Essential Commands

- `make` - build full application (UI + Go binary)
- `make build` - build Go binary only
- `make ui` - build UI assets only
- `make install` - install Go tools and dependencies
- `make install-ui` - install Node.js dependencies (`npm ci`)
- `make test` - run Go tests
- `make test-ui` - run frontend tests
- `make lint` - run Go linting (golangci-lint)
- `make lint-ui` - run frontend linting
- `npm run dev` - start Vue dev server (http://127.0.0.1:7071)
- `npm run playwright` - run integration tests
- `evcc --template-type [type] --template [file]` - test device templates
- `make docs` - generate template documentation

## Architecture Guidelines

### Core Components

- **main.go** serves as entry point and embeds web assets and i18n files
- **cmd/** contains CLI commands, application setup, and various utility commands (configure, detect, migrate, etc.)
- **core/** contains core business logic with main files (loadpoint.go, site.go) and subdirectories:
  - **loadpoint/** - EV charging point management modules
  - **planner/** - Smart charging planning algorithms
  - **coordinator/** - Multi-loadpoint coordination logic
  - **session/** - Charging session management
  - **vehicle/** - Vehicle-specific core logic
  - **soc/** - State of charge handling
- **api/** contains API definitions and types
- **server/** handles HTTP server, WebSocket, MQTT, database operations, and various handlers
- **charger/**, **meter/**, **vehicle/** contain device integrations
- **tariff/** contains tariff integrations
- **plugin/** implements plugin system for device and tariff communication
- **assets/** contains Vue.js frontend application

### Frontend Structure

- **assets/js/** contains the main TypeScript/Vue.js application with:
  - **views/** - Vue page components (App.vue, Config.vue, Sessions.vue, etc.)
  - **components/** - Reusable Vue components
  - **composables/** - Vue utility functions
  - **types/** - TypeScript type definitions
  - **utils/** - Utility functions
  - **mixins/** - Vue mixins
- **assets/css/** contains application stylesheets
- **assets/public/** contains static assets and metadata
- **i18n/** contains internationalization files
- **tests/** contains Playwright integration tests and test configuration files
- **dist/** contains built frontend assets (generated)

## Go Coding Standards

### Core Principles

- Follow Go idioms and conventions (Effective Go)
- Use `gofmt` for formatting, self-documenting names, early returns
- Handle all errors explicitly with meaningful messages
- Use interfaces for behavior contracts (small, focused, single responsibility)
- Use `context.Context` for I/O, long-running, or cancelable operations
- Organize code into logical packages with clear responsibilities
- Prefer composition over inheritance, minimize external dependencies

### File Patterns

- `_blueprint.go` - templates for new device implementations
- `_enumer.go` - generated enum code
- `*_decorators.go` - generated decorator pattern implementations
- Validate interface implementations: `var _ Interface = (*Type)(nil)`

### Error Handling

- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Use `errors.As` and `errors.Is` for type checking
- Use `errors.Join` for combining errors (prefer custom `joinErrors` helper)
- Create domain-specific error types (ClassError, DeviceError)
- Use `backoff.Permanent(err)` for non-retryable errors
- Implement panic recovery with `defer` and `recover()` in script contexts

### Testing & Code Generation

- Use `testing` package with `testify/assert` and `testify/require`
- Table-driven tests with struct definitions for multiple cases
- Use `gomock` for interface mocking, `go:generate mockgen` for generation
- Test both success and failure scenarios, use `require` for setup, `assert` for tests
- Use `go:generate` for code generation, regenerate after interface/enum changes
- Never manually edit generated files

### Context & Concurrency

- Use `context.Context` as first parameter for I/O operations
- Use `context.WithTimeout`, `context.WithCancel` appropriately
- Check `ctx.Done()` in long-running loops
- Propagate context through goroutines for proper cancellation
- Handle concurrent operations safely with Go's concurrency primitives

### Data Validation

- Filter `NaN` and `Infinity` values using `math.IsNaN()` and `math.IsInf()`
- Validate numeric inputs from external sources
- Use helper functions like `parseFloat()` that reject invalid values

## Vue.js/TypeScript Frontend Standards

### Core Architecture

- Use Vue 3 Options API (preferred over Composition API)
- Use reactive stores without Vuex/Pinia for cross-component state
- Use global app instance (`window.app`) only for: notifications (`raise()`), offline status (`setOffline()`/`setOnline()`), clearing notifications (`clear()`)
- Organize components by feature/domain in `assets/js/components/` subdirectories

### Component Development

- Use TypeScript for all new frontend code
- Use `const` instead of `function` for component methods (e.g., `const updateType = () =>`)
- Define TypeScript interfaces for component props, data, and API responses
- Implement accessibility features (tabindex, aria-label, keyboard handlers)
- Use descriptive names for variables, functions, and event handlers
- Use early returns for readability
- Use configured Axios instance for HTTP communication

### State Management

- Use `reactive()` from Vue for simple global state
- Implement property setters for nested object updates using helper functions
- Use localStorage with reactive wrappers for persistent settings
- Use Vue `watch()` for automatic persistence of settings changes
- Separate concerns with dedicated stores (settings, application state)

### TypeScript Patterns

- Define comprehensive interfaces for API responses and application state
- Use enums for constants (e.g., `THEME`, `CURRENCY`)
- Extend global interfaces for window object augmentation
- Use union types for flexible but type-safe configurations
- Use generic types for reusable utility functions
- Handle type assertions carefully with proper error handling
- Create focused utility functions with proper TypeScript typing

### Styling & Internationalization

- Use CSS Custom Properties for theming (semantic names: `--evcc-green`, `--evcc-battery`)
- Use existing custom media queries for responsive breakpoints
- Use `$t()` function for all user-facing strings
- Update both `i18n/en.json` and `i18n/de.json` for new strings
- Use hierarchical namespace: `{section}.{component}.{purpose}`
- Examples: `config.vehicle.titleAdd`, `main.vehicleStatus.charging`
- Action patterns: `titleAdd`, `titleEdit`, `save`, `cancel`, `delete`, `validateSave`
- Use placeholders for dynamic content: `{soc}`, `{duration}`, `{value}`
- Prefer context-specific keys over generic ones
- Test with German translations (20-40% longer text)

### Testing

- Write integration tests using Playwright for user workflows
- Use Storybook for component development and visual testing
- Use semantic selectors (roles, labels, button text); `data-testid` only when necessary
- Test error states and loading states

## Playwright Integration Testing

### Test Organization

- **Location**: `tests/` directory with `.spec.ts` files
- **Configuration**: `.evcc.yaml` files for different test scenarios
- **Utilities**: `tests/utils.ts` for common helpers, `tests/evcc.ts` for binary management
- **Categories**: `config-*.spec.ts` (UI config), `sessions.spec.ts`/`plan.spec.ts` (workflows), `smart-cost.spec.ts`/`limits.spec.ts` (features), `backup-restore.spec.ts`/`auth.spec.ts` (integration)

### Test Configuration

- Base URL: `http://127.0.0.1:7070`
- Parallel execution with different ports per worker for isolation
- Uses `./evcc` binary with test-specific configuration files
- Each worker uses isolated temporary database files
- Always runs with English UI language

### Essential Commands

- Must build before testing: `make ui build`
- Run tests: `npm run playwright` or `npx playwright test`
- Debug: `npx playwright test --debug`
- Specific test: `npx playwright test tests/config-loadpoint.spec.ts`

### Selector Strategy

- **Preferred**: Semantic selectors using `getByRole()`, `getByLabel()`, `getByText()`
- **Fallback**: `data-testid` only when semantic selectors aren't available
- **Examples**:
  - `page.getByRole("button", { name: "Add charger" })`
  - `page.getByLabel("Manufacturer").selectOption("Demo charger")`
  - `page.getByTestId("loadpoint")` (fallback only)

### Test Patterns

- Use test-specific `.evcc.yaml` configurations
- Import utilities from `tests/utils.ts` for common operations
- Focus on complete user journeys rather than isolated interactions
- Use `expectModalVisible()` and `expectModalHidden()` helpers
- Test configuration persistence across application restarts
- Standard structure: import `{ start, stop, baseUrl }` from `./evcc`, use `test.afterEach(stop)`
- Never use fixed timeouts, use existance of elements or wait for network idle

## Device Integration & Configuration

### Plugin System

- Device types: chargers, meters, vehicles, tariffs
- Plugin protocols: Modbus, HTTP, MQTT, JavaScript, Go
- Define device capabilities and configuration in templates at `templates/definition/[type]/`
- Test templates: `evcc --template-type [type] --template [file]`
- Update docs after template changes: `make docs`

### Configuration

- Use YAML format for all configuration files (default: `evcc.yaml`, or specify with `--config`)
- Provide clear validation and error messages for invalid configurations
- Support template-based device configurations with meaningful defaults
- Use SQLite as default database (default: `evcc.db`, or specify with `--database`) with proper migrations and data integrity

## Security & Performance Guidelines

### Security

- Validate all user inputs and sanitize data before database storage
- Use secure protocols (TLS) for external integrations
- Implement proper authentication and authorization
- Never log sensitive information (passwords, tokens, personal data)

### Performance

- Optimize database queries with appropriate indexes
- Handle concurrent operations safely with Go's concurrency primitives
- Implement proper caching strategies and connection pooling
- Avoid blocking operations in main application loop
- Include appropriate comments for complex business logic
