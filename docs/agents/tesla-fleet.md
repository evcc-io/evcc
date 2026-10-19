# Tesla Fleet

The `tesla-fleet` vehicle template connects evcc directly to the official Tesla Fleet API. The user brings their own Tesla developer app, and evcc acts as that app's backend. Setup, login and key pairing all happen in the evcc UI, without a terminal or a third-party service.

The older `tesla` template uses the same Fleet API with static tokens and an external command proxy (e.g. MyTeslaMate). Both render `type: tesla` (`vehicle/tesla.go`).

Generic building blocks (public origin, public routes, return to the starting page, readonly copy fields) are described in [provider-auth.md](provider-auth.md) and `templates/README.md`.

## Concepts

### The instance is the app's domain

A Tesla developer app is bound to one https domain. The allowed origin, the redirect uri and the domain hosting the command signing key must all match. evcc uses its public origin (`network.PublicOrigin()`) for all three: the remote access url, or an https external url. The template shows the origin and redirect uri as readonly fields to copy into the Tesla portal.

Because the domain is part of the app registration, changing the public origin later means updating the Tesla app and logging in again.

### Partner registration

Before the first login, Tesla requires the app's domain to be registered as a partner. evcc does this in a login hook, right before redirecting the browser to Tesla:

- It ensures the signing key exists.
- It registers the domain via `partner_accounts` with a client credentials token. Tesla fetches the public key from the domain during this call, so the domain must already be reachable.
- The account region is unknown before login, so registration runs against both regions (EU and NA). One success is enough.

Registration is idempotent and runs on every login.

### Signing key

Each evcc instance has one P-256 key, stored in the settings table. The public key is served on a public route at `/.well-known/appspecific/com.tesla.3p.public-key.pem`.

Tesla binds a public key to one app and domain. If registration fails because the key is already taken (other app or old domain), evcc generates a new key and registers once more. Vehicles paired with the old key must pair again.

### Region

After login, evcc asks the Fleet API for the account region. All data and command requests go to that regional host. Tokens are requested for the NA audience, which is valid for both regions.

### Commands

The two modes differ in how commands reach the vehicle:

| Mode        | Selected by                        | Tokens                   | Commands                            |
| ----------- | ---------------------------------- | ------------------------ | ----------------------------------- |
| Interactive | client id + secret, no tokens      | UI login (provider auth) | signed in-process (vehicle-command) |
| Static      | access and refresh token in config | from config              | external command proxy              |

In interactive mode, evcc signs commands itself with Tesla's `vehicle-command` library. The vehicle only accepts them once the user has added evcc's key to the car ("virtual key"). The template offers the pairing link `https://tesla.com/_ak/<domain>` as a readonly advanced field. Pairing is only needed for vehicle-side charge control, e.g. with a Tesla Wall Connector. Data access works without it.

Vehicles without command protocol support (pre-2021 Model S/X) fall back to plain REST commands.

## Code map

| Area                          | Location                                        |
| ----------------------------- | ----------------------------------------------- |
| Vehicle, mode selection       | `vehicle/tesla.go`                              |
| Fleet API client, region      | `vehicle/tesla/api.go`, `fleet.go`              |
| OAuth and login hook          | `vehicle/tesla/oauth.go`                        |
| Partner registration          | `vehicle/tesla/partner.go`                      |
| Signing key, public key route | `vehicle/tesla/key.go`                          |
| Signed commands               | `vehicle/tesla/signed.go`                       |
| Pairing link service          | `vehicle/tesla/service.go`                      |
| Template                      | `templates/definition/vehicle/tesla-fleet.yaml` |

## Self-hosting without remote access

Remote access is the documented path: it provides the public https domain and forwards the public routes automatically. Users who run their own public reverse proxy can use the same in-app setup:

1. Make evcc reachable at a public https domain with a certificate from a public CA, e.g. `https://evcc.example.com`. Tesla rejects plain http and self-signed certificates.
2. Forward at least these paths to evcc without extra authentication:
   - `/.well-known/appspecific/com.tesla.3p.public-key.pem`: Tesla fetches the public key here during login.
   - `/providerauth/callback`: the browser returns here after the Tesla login.
3. In evcc, open "Network" in the configuration and set "External URL" to the public address. It must start with `https://`.
4. Add the vehicle "Tesla". The "requires Remote Access" hint can be ignored. The copy fields now show the public address. Continue with the setup steps in the template description.

The setup itself can be done from any address, e.g. `http://evcc.local:7070`. After the Tesla login, the browser is sent back to the page the login started from.

Changing the external url later requires updating the Tesla app and logging in again. If the key was registered for the old domain, evcc replaces it and vehicles must pair again.

## Follow-ups

- Multiple Tesla accounts per client id (needs an account discriminator in the `plugin/auth` subject).
- Powerwall template on the same OAuth.
- User docs on docs.evcc.io.
