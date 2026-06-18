# API Security & Authentication

How evcc authenticates HTTP requests and how endpoints are classified by
sensitivity.

## Threat Model

evcc is designed for use within a trusted home network. The auth layer
protects credential management, configuration changes, and system operations
(logs, backup/restore/reset, shutdown). Read-only state and basic charging
controls are intentionally unauthenticated.

## Auth Modes

| Mode       | Trigger               | Behavior                                          |
|------------|-----------------------|---------------------------------------------------|
| `Enabled`  | default               | password required; JWT or API key accepted        |
| `Disabled` | `--disable-auth` flag | all auth checks skipped                           |
| `Locked`   | demo mode             | mutating endpoints return 403; reads still work   |

Mode is fixed at startup. The frontend mirrors the mode so admin actions can
be greyed out and a banner shown.

## Endpoint Sensitivity Tiers

Three tiers, by what the caller has to prove:

**Public.** No auth. State, loadpoint controls, login. Anyone on the
network can read and operate.

**Secure.** Requires a valid session: either the auth cookie (browser, JWT)
or an API key in the `Authorization: Bearer …` header (automation). Used
for configuration and system administration.

**Critical.** Secure plus an additional admin-password check inside the
handler. Used for destructive or credential-scoped operations.

For some Critical endpoints (backup, restore, reset) the password check is
**skipped when the caller is authenticated via API key**, so unattended automation
doesn't need to embed the admin password. For credential-management
endpoints (rotate API key, change admin password) the password check is
**strict**: a leaked API key must not be able to rotate itself or change
the admin password.

Disabling auth short-circuits all checks.

## Sessions

Two transports, no overlap:

- Browsers use a session cookie (JWT, 90-day TTL, issued on login).
- Automation uses an API key in the `Authorization: Bearer …` header.

API keys are random alphanumeric strings prefixed `evcc_`. The prefix makes
leaked keys recognizable to secret-scanning tools.

A single API key per installation; regenerating replaces the previous one.
Plaintext is shown to the user **once** at generation time and cannot be
retrieved afterwards.

## Credential Storage

Admin password and API key are stored as bcrypt hashes. The JWT signing
secret is a per-installation random value. Plaintext credentials are never
persisted.

Removing the admin password (CLI recovery) also clears the JWT secret and the
API key, which invalidates all outstanding sessions and any previously-issued
API key. Regenerating the API key replaces the stored hash; the previous key
stops working immediately.

## API Key Lifecycle

Two operations:

- **Status.** Whether a key is configured. Secure tier; never returns
  plaintext.
- **Regenerate.** Critical tier, strict password check. Returns the new
  plaintext key exactly once.

There is no delete operation: regenerating and discarding the new key
achieves the same effect (the previous key stops working immediately).

## Endpoint Matrix

| Endpoint category                            | Tier      | Additional Requirements            |
|----------------------------------------------|-----------|------------------------------------|
| State / read-only / basic charging control   | Public    |                                    |
| Set or update admin password                 | Public    | admin password                     |
| Configuration                                | Secure    |                                    |
| System: logs, cache, shutdown                | Secure    |                                    |
| API key status                               | Secure    |                                    |
| System: backup / restore / reset             | Critical  | api key or admin password          |
| API key regenerate                           | Critical  | admin password                     |

**Public** endpoints accept any caller. **Secure** endpoints require a
valid session (cookie or API key). **Critical** endpoints require extra
authentication in the form of an admin password (or, for some, an API
key).

## Frontend

Auth management lives under **General Config → Security**, which links to
two sub-flows: change admin password, and manage the API key. The API key
flow has a reveal view that shows the plaintext exactly once with a
copy-to-clipboard link.

When auth is disabled, the security modals show a warning banner and
disable all action buttons. This is UI-only; the backend still accepts the
underlying calls so legitimate automation against a disabled-auth instance
keeps working.

## OpenAPI

The OpenAPI spec declares two security schemes (cookie and bearer);
protected operations accept either.
