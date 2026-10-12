# Discovery Hints

Routine for refreshing the embedded vendor registry and the `discovery` hints of device templates. The hint format and the guidelines are documented in the `discovery` section of `templates/README.md`, the runtime side in [Plugin System](plugin-system.md).

## 1. Update the vendor registry

`util/discovery/oui.txt.gz` is built from the IEEE registries and is never downloaded during a build. Each line is `PREFIX<tab>Organization`, with 6 (MA-L), 7 (MA-M) or 9 (MA-S) hex digits.

```sh
gunzip -c util/discovery/oui.txt.gz > /tmp/oui.old.txt
make oui-update
gunzip -c util/discovery/oui.txt.gz > /tmp/oui.txt
diff /tmp/oui.old.txt /tmp/oui.txt | grep '^[<>]'
```

The diff lists new (`>`) and removed (`<`) assignments. A removed prefix that is still used as a hint has to be checked in step 2.

## 2. Templates with `mac` hints

Resolve every hinted prefix to its organization. Unregistered prefixes are reported.

```sh
for p in $(grep -rhE '^  mac:' templates/definition | grep -oE '[0-9A-F]{6,9}' | sort -u); do
  grep "^$p	" /tmp/oui.txt || echo "$p	UNREGISTERED"
done
```

The separator after `$p` is a literal tab. For each organization, list all its assignments and compare them with the templates of that vendor. Legal names vary (`SMA Solar Technology AG`, `KOSTAL Industrie Elektrik GmbH`), so search for the distinctive part.

```sh
grep -i 'sma solar' /tmp/oui.txt
grep -rl '0015BB' templates/definition
```

Prefixes of the organization that no template lists are candidates.

## 3. Templates without hints

These templates have a `host` param but no `discovery` section.

```sh
grep -L '^discovery:' $(grep -l '^  - name: host' templates/definition/*/*.yaml)
```

Search the registry for the manufacturer, starting from `products.brand` in the template. Brands often differ from the registered legal entity, so the parent company or the vendor's imprint may be needed.

## 4. Home Assistant cross-check

Home Assistant aggregates the discovery matchers of all integrations into generated files. No scan of the project is needed.

```sh
base=https://raw.githubusercontent.com/home-assistant/core/dev/homeassistant/generated
curl -sSfL $base/dhcp.py > /tmp/ha-dhcp.py
curl -sSfL $base/zeroconf.py > /tmp/ha-zeroconf.py
grep -n -B4 -A4 -i 'fronius' /tmp/ha-dhcp.py /tmp/ha-zeroconf.py
```

Match the `domain` values against evcc template names and brands. Domains are snake case and not always the brand (`powerwall`, `tesla_wall_connector`, `enphase_envoy`, `kostal_plenticore`).

| Home Assistant                                 | evcc                           |
| ---------------------------------------------- | ------------------------------ |
| `"hostname": "sma*"`                           | `hostname: ["sma*"]`           |
| `"macaddress": "0015BB*"`                      | `mac: ["0015BB"]`              |
| `"_shelly._tcp.local."`                        | `mdns: ["_shelly._tcp"]`       |
| `"_http._tcp.local."` with `"name": "shelly*"` | `mdns: ["_http._tcp:shelly*"]` |

Entries with only `registered_devices` carry no hint. `ssdp.py` is not used, evcc has no SSDP hints. The `HOMEKIT` block in `zeroconf.py` is irrelevant.

## 5. Applying candidates

The guidelines in `templates/README.md` apply. In addition:

- Vendors with a broad product range (Huawei, Samsung, TP-Link, AVM) and network module makers (Espressif, Texas Instruments, Murata) are skipped for `mac`. Their prefixes match phones, routers and unrelated devices. `mdns` or `hostname` hints are used instead.
- Generic hostnames from Home Assistant (`target`, `espressif`) are not adopted.
- All templates of one device family get the same hints.
- `mac` values are 6 to 9 upper case hex digits without separators. `mdns` values are the service type without `.local.` in the announced spelling (`_go-e_go-eCharger._tcp`), optionally followed by `:` and an instance name pattern.
- Candidates that cannot be tied to the device family with reasonable certainty are reported, not applied.
- A hint that was removed deliberately is not added again. The history of the candidate value and of the template shows earlier removals and their reason.

```sh
git log --oneline -S'0015BB' -- templates/definition
git log -p --follow -- templates/definition/meter/sma-hybrid.yaml | grep -B3 -A3 '^-.*\(mdns\|hostname\|mac\):'
```

## 6. Verify and report

```sh
go test ./util/templates/...
```

The report lists the registry changes in one line (count of new and removed assignments) and one row per changed template: template, added or removed hint, source (IEEE organization or Home Assistant domain). Open candidates follow as a separate list with the reason for not applying them.
