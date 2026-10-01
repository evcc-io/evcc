# Provider Authentication

How evcc connects to third-party accounts (vehicle clouds, Home Assistant, heat pump clouds) from the UI: the mechanisms, the flows a user can encounter, how connections are shared, and what to pick for a new integration.

This is about logging evcc into an external provider. Logging users into evcc itself is covered in `api-security.md`.

## Building blocks

### Backend

- `api.AuthProvider` is the contract for a login-capable provider: start a login, finish a redirect callback, log out, report the authentication state, and give a display name. `api.AuthChallenger` is the optional extension for logins that run inside evcc with stored credentials and need one piece of user input on the way. A challenge has a kind that selects the label of the single answer field (captcha or code), an optional image (the captcha) and an optional link (a vendor page the user opens to obtain the answer). Submitting an answer returns the next challenge or nothing when the login completed; a wrong answer comes back as a fresh challenge.
- `server/providerauth` is the process-wide registry of providers. It exposes login, logout, submit (challenge answer) and the public OAuth callback, and it publishes the `authProviders` websocket key that drives every list, badge and modal in the UI. The login response has three shapes: a redirect url, a device code, or a challenge.
- `plugin/auth` is the registry of `oauth2.TokenSource` factories. The generic OAuth implementation derives a stable provider id from the client configuration, persists the token in the settings table under that id, registers with `providerauth`, and hands the same instance to every device using the same configuration. Vendor-specific auth types wrap it or implement the contracts themselves.
- The config API has an auth check endpoint: given an auth type and its params it instantiates the token source and reports "connected" or "login required" together with the provider id. Instantiating is a side effect: the provider is listed in the UI from that moment on, whether or not a device is saved later.
- Devices do not block on login. They ask the token source on every poll, receive a login-required error until the user connects, and work from the next poll on. No restart is involved in any login.

### Template

A template opts into the UI login flow with an `auth` block naming the auth type and the params it needs. Those params are ordinary template params (client id, redirect uri, credentials, ...) and are rendered into the device config like any other. Service values can resolve the callback url of this instance or fetch vendor data that only exists after login. Requirements render as prerequisite hints above the form.

### Frontend

Two surfaces, sharing the same components for the three login shapes (connect link, device code display, challenge form):

- The device modal renders a two-step form for templates with an `auth` block. Step 1 shows only the auth params and a prepare button that runs the auth check. Depending on the answer it shows the connect link, the device code, or the challenge. Step 1 is skipped when the provider is already connected, and it auto-runs whenever all required auth values are present. Step 2 is the regular device form.
- The provider modal, reachable from the "Authorization Status" card on the config page, from the hamburger menu and from the callback banner, runs a login or a logout without device context. It starts the login on open.

The callback of a redirect login lands back on the config page with a success or error banner and reopens the device modal that started it.

## Flow types

|     | Flow                                | Where the user enters credentials                                                                                   | evcc reachable from the browser                                         | Prerequisites outside evcc                                                                                       | UI support |
| --- | ----------------------------------- | ------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------- |
| A   | OAuth redirect (authorization code) | Vendor login page in a new tab                                                                                      | Yes, at the registered redirect uri (some vendors require public https) | Developer app with client id/secret and redirect uri, unless the vendor derives the client from the instance url | Yes        |
| B   | OAuth device code                   | Vendor page, user types the code shown by evcc                                                                      | No                                                                      | Client id from the vendor portal                                                                                 | Yes        |
| C   | Challenge (server-side login)       | Credentials are template auth params, evcc replays the vendor login and shows the captcha or asks for a pasted code | No                                                                      | None                                                                                                             | Yes        |
| D   | Paste tokens                        | External website or script, tokens copied into params                                                               | No                                                                      | Third-party token tool                                                                                           | No         |
| E   | CLI token                           | Terminal, `evcc token` prints a url and asks for the code                                                           | No                                                                      | Shell access to the host                                                                                         | No         |
| F   | Headless credentials                | Template params `user` and `password`, evcc logs in unattended                                                      | No                                                                      | None                                                                                                             | No         |

D, E and F have no login UI; failures show up as device errors. F breaks as soon as a vendor adds a captcha or bot protection. C is F with the captcha surfaced in the UI. E is C done in a terminal.

## Screens

Flow A from the device modal, new device:

1. Select the product. Description and prerequisite hints appear, then only the auth fields and "Prepare connection".
2. Prepare connection. Either step 2 opens directly (already connected) or a "Connect to vendor.com" link appears. Errors from the token source render inline.
3. Vendor login in a new tab, callback to evcc.
4. Config page with a success banner (or an error banner). The device modal reopens at step 2.
5. Fill the remaining fields, validate, save.

Flow B differs in step 2: the modal shows the user code and its expiry next to the link, and completion arrives via websocket without a callback.

Flow C differs in step 2: prepare runs the login with the entered credentials. A wrong password shows inline. If the vendor asks for a captcha, the image and an answer field replace the connect button; a wrong answer returns a fresh image. Success switches to step 2 directly.

Editing a connected device skips step 1. Editing a device whose token expired shows step 1 again with the login prepared.

Without device context, the card or the hamburger menu opens the provider modal, which prepares the login on open and shows the matching shape. After the websocket reports the provider as authenticated, the modal shows a success message. Disconnect is a confirm dialog in the same modal.

## Cardinality and lifecycle

- One connection per distinct auth configuration. For the generic OAuth implementation that is the client configuration, so all devices using the same developer app share one login and appear together in the display name. Two accounts at the same vendor need two developer apps. Vendor-specific providers choose their own key, typically one connection per account, with the account in the display name because the published map is keyed by it.
- Tokens live in the settings table under the provider id, not in the device config. Deleting or reconfiguring a device leaves the token and the provider entry behind until restart. Nothing unregisters a provider.
- The auth check registers a provider as soon as the user prepares a connection. Cancelling the modal leaves the entry in the card until restart.
- A redirect uri or client id entered as a param is persisted and rendered into the device config, even where it only mirrors a service value.

## Known gaps

- Redirect flows require the browser to reach evcc at the redirect uri. Local http works for most vendors, some require public https.
- Vendors with a browser login but a fixed redirect (custom app scheme, void url) can only be completed by pasting the code or the redirect url back into evcc. The challenge contract covers this (link plus code answer), the CLI and third-party tools do it today, no provider uses it in the UI yet.
- Some template descriptions still document manual procedures (token tools, terminal commands) for vendors that could use one of the UI flows.
- No cleanup of providers or tokens on device deletion or config change.
- Users meet several mental models for "connect my account" (A to F) and several places to read about them (template description, docs site, CLI).

## Testing

The flows are tested end to end without any vendor. The `demo` auth type in `plugin/auth` plays the provider and switches its behaviour by a `method` param: `redirect`, `device-code`, `challenge` (credentials plus a fixed captcha) or `code` (link to the mock login page, pasted code or redirect url). Test templates under `tests/` use it in an `auth` block and are passed to evcc via `--template`, so the real device modal, provider modal and card are exercised. The test simulator serves the mock vendor login page for the redirect flow and the data the device validation reads. When a login shape or a UI surface changes, extend those specs rather than testing against a vendor.

## Guidance for new integrations

Pick the first option the vendor supports:

1. OAuth device code: flow B. Preferred, needs no reachable url.
2. OAuth with a user-registrable app and a configurable redirect uri: flow A with the generic OAuth implementation and a service value for the callback url.
3. Browser login with a fixed redirect: flow C with a challenge that carries the login link and asks for the pasted code. Captcha and two-factor handling stay in the vendor's own page and evcc never sees the password.
4. Only a scripted login with captcha: flow C with credentials as template auth params and a captcha challenge. The scraping breaks whenever the vendor changes the page, the challenge contract does not.
5. Only a long-lived API token exists: plain masked param, flow D. Document where to get it in the template description.

Rules that keep the model in one piece:

- Every provider is declared through the template `auth` block and shows up in the "Authorization Status" card. No login that only exists after a device is saved.
- Everything user-facing goes through `providerauth` and the two existing UI surfaces. No per-vendor endpoint, modal branch, or form definition.
- Starting a login must be idempotent. The UI prepares on template load, config load and modal open, so a pending challenge is re-served, not restarted.
- A challenge carries only what the vendor page cannot deliver otherwise. Anything the user knows upfront is a template param.
- Login never requires a restart. Providers are constructed lazily and pick up the token on the next poll. Persist the token immediately so a restart right after login does not lose it.
- Do not add flow F templates for cloud vendors with bot protection.
