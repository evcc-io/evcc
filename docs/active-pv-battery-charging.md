# Active PV Battery Charging: journal-based control

Active PV battery control can issue real sonnen charging commands. It is no longer observation-only.
The hardware lease design remains available for other adapters, but sonnen now uses
an action journal under the explicit assumption that evcc is the intended sole controller.

## Observation and decisions

The sonnen adapter reads mode, charging flag, power and SoC from one uncached
`/api/v2/status` response. Mode 1 is manual; modes 2 and 10 are automatic.
Missing or unknown mode, non-OnGrid status, invalid measurements and observations
older than ten seconds prevent acquisition. Repeated identical timestamps do not
refresh freshness. Timestamps are change markers: the first response cannot be
independently dated. Configured identity is `sonnen:<host>`, not a hardware serial
number. Changing the configured host causes a conflict; swapping hardware at the
same address cannot be detected by this identity.

Automatic native charging is never taken over, even if export persists. The native charging flag always prevents takeover. With the flag false, only
charging above 5 W is treated as native charging. This conservative idle tolerance
addresses the observed sonnen status `BatteryCharging=false`, `Pac_total_W=-3`
on 11 October 2026; it is not a manufacturer accuracy specification. Idle watts
are not added to the initial charging budget. The configurable start value is
independent of this tolerance. Logs include the observed `nativeCharging` flag.
An unexplained manual mode produces a persistent conflict without writing to the battery.

Start requires the configurable GUI **PV charging start value**, default 500 W,
automatic idle operation, sufficient SoC headroom, valid PV measurements and an
available positive charge-power limit. The start value is independent of that limit.
The 100 W reserve (or larger site reserve) is deducted after the threshold check:
500 W export initially requests at most 400 W charging. Other battery discharge
and newly allocated loads are excluded. Household demand is already in grid power.

During charging, measured charging power plus usable export minus reserve sets
the new request, capped by PV and the configured limit (currently 3300 W).
Between zero grid power and the export reserve, the last request is held.
Falling below the start value alone does not stop charging. Any positive valid
measured grid import, full battery/SoC limit, invalid budget, missing PV or revoked
permission starts return to automatic control. There is no arbitrary 80% taper.
This is cycle-based regulation, not a guarantee against short physical import transients.

## Durable action journal

Each battery has an append-only JSONL file next to the configured database:
`evcc.db.pv-<SHA256 of device config name>.jsonl`. Its `.lock` file holds an OS
exclusive lock for the process lifetime. Another process using that same journal
cannot claim it. Independent databases/hosts do not share this protection.
In-memory databases or unavailable journal storage disable journal-based control.
No database migration is added. The existing start-value setting remains in evcc settings.

Every command follows this order:

1. Append and sync intent (identity, previous automatic mode, requested watts,
   revision, timestamp and reason), including directory sync on Linux.
2. Send the battery command.
3. Enter `confirming` and observe in later control cycles (90-second confirmation
   window). Accept only a valid sample newer than the command; check a setpoint
   readback when supported. Old automatic mode or a previous setpoint is initially
   pending, not evidence of foreign control. Do not resend charge while pending.
4. Append and sync confirmation. An unchanged command is not rewritten each cycle.

For devices without a charge-setpoint readback, confirmation additionally requires
measured charging above the 5 W idle tolerance. A manual mode response alone is
insufficient. This proves charging activity, not exact acceptance of the requested
watts. A pending command without charging times out after 90 seconds and returns
control. If charging disappears during an active session without readback, a
separate 90-second timer prevents subsequent requests or the grid deadband from
masking that loss indefinitely. Grid import still starts release immediately in
the next control cycle, without waiting for either timeout.

The optional `BatteryChargePowerUpdater` on a charge controller changes a setpoint
only after the journal has established and freshly observed its own manual control.
The sonnen `chargepowerupdate` plugin sends only `POST /api/v2/setpoint/charge/{watt}`.
Acquisition retains the existing mode/discharge-zero/charge sequence. Devices
without this optional update path retain their existing setter. Restart recovery,
unknown manual control and failed update writes retain the journal's restore rules.

On 12 October, the supplied hardware logs showed 410 W requested and 409-414 W
charging, followed by 432/400 W requests with only 2-3 W actual charging. A later
482 W request produced 481 W actual charging. The previous adapter repeated the
manual-mode command and a discharge-zero command on every update. Removing those
intermediate writes eliminates an avoidable interruption opportunity; their role
in the observed hardware gap is not yet proven. The same logs confirmed release
after 10 W grid import and automatic mode 2 in the next cycle. The revised update
path still requires a controlled hardware test.

Release and user decisions are persisted the same way. A torn final record,
unreadable file blocks automatic control. A write failure blocks new charging,
but a previously validated durable own command still authorizes best-effort
rollback even if the journal can no longer be written. Corrupt records are not
silently discarded. The files contain no token. History currently has no automatic
rotation; monitor disk use. An evcc database export does not include these sidecar
files. Copy journal files with the stopped service when preserving recovery history.
If a journal is absent while the battery is manual, evcc reports a conflict.

## Restart and fault state machine

| Journal / observed state | Action |
| --- | --- |
| No active record, automatic idle | Observe; acquire only when all start conditions hold. |
| Automatic native charging | Leave it alone. |
| Confirmed active record after restart | Restore previous automatic mode before any new charging. |
| Own intent, confirmation pending | Wait for later samples; abort immediately on unsafe site conditions. |
| Own intent, confirmation deadline exceeded / command error | Restore previous automatic mode; no conflict solely for a delayed reply. |
| Own intent after restart | Restore previous automatic mode even if an old automatic sample is still visible. |
| Unknown manual mode / changed identity / different readable setpoint | Persist conflict; send no further commands. |
| Battery returns to automatic mode | End our session; this alone is not foreign control. |
| Read failure while active or confirming | Persist recovery; retry observation and release after reconnection. |
| Release fails | Keep recovery pending; do not resume charging. |
| Journal unavailable on startup | Block writes, show the problem. |
| Journal write fails after own durable intent | Block new charging and attempt rollback; keep recovery pending. |
| User disables control | Persist disabled state across restarts; no device write. |

A conflict is sticky across restart and is shown on the Battery page. The user can
choose **Yes / unsure: disable** or **No: allow evcc recovery**. Recovery permission
is persisted first and executed on the next cycle, after another observation.
It restores the recorded automatic mode (2 or 10), or the configured automatic
default when no previous mode is known. It does not immediately start charging.
The decision endpoint requires authentication and the displayed journal revision;
a stale dialog receives HTTP 409 and cannot approve a newer conflict.


A release is sent once, then checked on later observations. The first still-manual
sample is not an error. If restoration remains unconfirmed, retry at intervals of
at least 30 seconds. Recovery never expires into permission to charge. The normal
30-second site cycle determines observation latency; network import is checked
while charge confirmation is pending and takes precedence over its deadline.
A graceful shutdown waits up to 15 seconds for later restoration confirmation
(subject to device I/O and the process shutdown deadline). Unconfirmed restoration
is logged explicitly and retained for the next start. Process death or network
failure cannot guarantee immediate physical restoration.

The first confirmed manual mode is not proof that measured watts already match.
Dynamic updates likewise wait for later confirmation before sending another
charge request. Outstanding commands reserve their requested budget, even though
the active controller target is zero during confirmation/recovery.

Conflicts from old builds remain locked; the update does not silently discard
previous user decisions or overwrite unknown manual control. The October 11
hardware incident was returned to automatic mode explicitly by the user. Its old
conflict can be cleared through the existing authenticated recovery decision,
which first reconciles automatic operation and does not charge in that cycle.

Logs distinguish `controllerTarget` from `lastChargeRequest`. A zero controller
target does not mean a zero-watt command was sent. `lastObservedMode`,
`lastObservedPower`, `observationValid` and `observedAt` disclose observation age.
Even conflict/disabled sessions refresh read-only observations; failed reads mark
last-known values invalid in the GUI.

## Limits of the ownership evidence

The journal proves what evcc recorded and attempted, not exclusive hardware ownership.
The sonnen adapter has no verified setpoint readback, owner token or independent
command watchdog. HTTP success plus manual-mode observation is recorded honestly
as acknowledgement, not as verification of actual requested watts. Another writer
changing a setpoint while remaining in manual mode is currently undetectable.
An optional `ChargeSetpoint` readback is supported generically and tested, but measured
battery power must never be substituted for it: taper, limits and response lag
naturally change actual watts.

A powered-off or disconnected evcc cannot stop a previously issued manual charge.
Recovery runs when evcc and communications return. Battery firmware protections
remain in effect, but are not a substitute for a verified independent watchdog.
A read/write race against another controller is not fenced by the local journal.

## EV coordination and existing mode features

Manual or unobserved batteries contribute no EV discharge capacity. Their existing
charging demand is not advertised as reclaimable PV. For journal-capable batteries,
owned charging is released when a waiting fixed-phase EV has enough PV to start
or a charging EV can use more PV. Pending vehicle and battery requests reserve
their unmeasured demand so both controllers cannot spend the same export.
The existing site buffer SoC governs support for a running EV; SoC alone cannot
start it. Explicit EV boost on other sites retains its existing permission guard.
The cloud-boost controller itself sends no manual discharge commands; see
[pv-cloud-boost.md](pv-cloud-boost.md).

Other site-level battery commands still do not bypass the ownership guard:
legacy tariff charging, external mode requests and discharge holds are suppressed
for guarded devices. These features have not been migrated to the journal.
`batteryMode` is a requested site mode; use `batteryPVControl` for actual observations.
This restriction is material when assessing a controlled test of this development build.

## Verification and ARM64 build

Tests cover write ordering, confirmed and interrupted restart paths, native charging,
conflict decisions, stale revisions, failed observations/writes/releases, identity
changes, measured-power versus setpoint differences, torn journals, file locking,
SoC/grid/start-value boundaries and EV budget exclusion. HTTP tests exercise sonnen
status and exact restoration of modes 2/10, plus refusal of arbitrary/manual restore
modes. GUI tests verify explicit decisions, persistent errors and disabled state.
These are simulated tests, not live hardware validation.

```sh
go test ./core/... ./api/... ./server/... -count=1
go test ./meter -skip '^TestTemplates$' -count=1
go test ./meter -run '^TestTemplates/sonnenbatterie' -count=1
vp test run --maxWorkers=1 --no-file-parallelism
vp build
mkdir -p build
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags=release -trimpath \
  -ldflags="-X github.com/evcc-io/evcc/util.Version=0.316.2-pv-confirmation-dev -X github.com/evcc-io/evcc/util.Commit=$(git rev-parse --short HEAD) -s -w" \
  -o build/evcc-pv-confirmation-linux-arm64 .
sha256sum build/evcc-pv-confirmation-linux-arm64
```

The full meter template matrix has unrelated platform-specific connection error
assertions on Windows; the sonnen template is checked separately. Building does
not install anything. Production deployment remains a separate controlled step.
