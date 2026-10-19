# Tesla Fleet Integration

The `tesla-fleet` vehicle template connects evcc directly to the official Tesla Fleet API with the user's own Tesla developer app. Onboarding happens entirely in the evcc UI, without a terminal or third-party proxy. It complements the older `tesla` template, which uses static tokens and a command proxy (e.g. MyTeslaMate).

## Modes

Both templates render `type: tesla` (`vehicle/tesla.go`). `FleetConfig.Interactive()` selects the mode:

| Mode        | Config                        | Tokens                      | Commands                                 |
| ----------- | ----------------------------- | --------------------------- | ---------------------------------------- |
| Interactive | client id + secret, no tokens | OAuth login in the UI       | signed in-process (`SignedController`)   |
| Static      | client id + access/refresh    | from config (`NewIdentity`) | via command proxy (`Controller` + proxy) |

Both modes read data with the inlined Fleet API client (`vehicle/tesla/api.go`). `FleetConfig.Client()` resolves the account region via `/users/region` and returns an `*API` bound to the regional base url.

## Interactive onboarding

1. The template shows two readonly copy fields, _Allowed Origin URL_ and _Allowed Redirect URI_ (services `auth/origin?public`, `auth/redirecturi?public`). The user enters them in the Tesla developer portal and copies the client id and secret back.
2. Login uses the generic provider auth (`plugin/auth`, see [provider-auth.md](provider-auth.md)). Auth params are the client id and secret. The redirect uri is `<origin>/providerauth/callback`.
3. A login hook runs before the redirect. It ensures the signing key exists and registers the origin host as partner domain via `partner_accounts` in both EU and NA, since the account region is unknown before login. One successful region is enough.
4. After login, `/users/region` picks the regional Fleet API host for data and commands.

The token request uses `audience` without a trailing slash (`FleetBaseUrl`, `FleetBaseUrlEU`).

## Signing key and public path

- One evcc instance has one P-256 key, stored in settings as `tesla.privateKey` (`vehicle/tesla/key.go`).
- Tesla fetches the public key from `https://<origin host>/.well-known/appspecific/com.tesla.3p.public-key.pem` during partner registration. The path is registered with `service.RegisterPublic`, mounted on the root router in `server/http.go`, without authentication.
- The remote access tunnel lets public routes and `network.CallbackPath` through without remote credentials (`server/remote/tunnel.go`), because Tesla and the user's browser reach them directly.
- Tesla binds a public key hash to one app and domain. On `Public key hash has already been taken`, evcc rotates the key and registers again once. Vehicles must then pair the new key.
- Pairing link: readonly advanced param `virtualkey` (service `tesla/virtualkey`), `https://tesla.com/_ak/<origin host>`. Pairing is only needed for vehicle-side charge control, e.g. with a Tesla Wall Connector.

## Commands

- `SignedController` (`vehicle/tesla/signed.go`) uses `teslamotors/vehicle-command`: token from `API.TokenSource()`, host from `API.Host()`. Nominal errors (already charging, etc.) are ignored.
- Vehicles without protocol support (pre-2021 Model S/X) fall back to the REST `Controller` against the Fleet API.
- `vehicle-command` derives the host from the token audience. The interactive login only requests the NA audience, so the host is overridden with the region host.

## Provider auth details

- The UI sends the full page url as `return` on `providerauth/login`. The handler only accepts it on the host the login was requested from and appends `callbackCompleted`/`callbackError`. Users on `evcc.local` land back on `evcc.local` with their session.
- The callback validates the state first and answers a plain 400 otherwise.
- Entered form values are stashed across the external login (`assets/js/utils/authStash.ts`), because the callback may return in a new tab.

## Generic template mechanisms

- `readonly: true` params: display-only copy fields fed by `service:`. They are listed in `auth.params`, never posted or persisted, and excluded from docs and required validation.
- Requirement `remoteaccess`: shows `RemoteAccessRequired.vue`, a warning linking to the remote access modal. It does not block saving.

## Public origin

`network.PublicOrigin()` (`server/network/service.go`) is the single source for the origin:

1. Remote access url, if remote access is enabled.
2. External url, if explicitly configured as https.
3. Remote access url of a disabled remote access, so devices set up through it keep working.

The `?public` copy fields follow steps 1 and 2 only and stay empty otherwise. `FleetConfig.Origin` overrides it for yaml configs, but the template does not expose it.

## Using without remote access

Remote access is the documented path. Users with their own public reverse proxy can use the same in-app flow:

- **External url:** set `network.externalUrl` to the public https address, e.g. `https://evcc.example.com`. http and the internal url fallback are ignored.
- **Valid certificate:** Tesla only accepts https with a CA-signed certificate for non-localhost origins.
- **Public paths:** the proxy must forward these without authentication:
  - `/.well-known/appspecific/com.tesla.3p.public-key.pem`, fetched by Tesla during partner registration
  - `/providerauth/callback`, the browser redirect target after login
- **Host header:** the proxy should keep the original `Host` header. Otherwise the return url check fails and the callback falls back to `/#/config`, losing the open modal.
- **Setup via public url:** configure the device while browsing evcc on the public url, so login and callback run on the same host.
- **Ignore the warning:** the `remoteaccess` warning still shows by design (advanced use case). It never blocks the setup.
- **Origin change:** changing the external url later requires logging in again, and may rotate the key so vehicles must pair again.

## Follow-ups

- Multiple Tesla accounts per client id (needs an account discriminator in the `plugin/auth` subject).
- Powerwall template on the same OAuth.
- User docs on docs.evcc.io.
