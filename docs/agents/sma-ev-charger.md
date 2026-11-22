# SMA EV Charger and eCharger

SMA ships two generations of AC chargers that share one parameter model:

- **EV Charger** (`EVC7.4-1AC-10`, `EVC22-3AC-10`), firmware 1.2.x, hardware rotary switch (smart/boost).
- **eCharger** (`EVC22-3AC-20`), firmware 2.x, ennexOS based, no rotary switch.

All protocols expose the same SMA channels (`Parameter.*`, `Measurement.*`) with the same enum codes. Shared channel ids and codes live in `charger/sma`; protocol drivers only implement transport and control.

| Protocol                            | Driver                    | Template                    | Hardware                |
| ----------------------------------- | ------------------------- | --------------------------- | ----------------------- |
| REST (`/api/v1`, OAuth token)       | `charger/sma-rest.go`     | `smaevcharger` (deprecated) | EV Charger, eCharger    |
| SEMP (`/SEMP`, XML)                 | `charger/semp.go`         | `semp-sma`                  | EV Charger, eCharger    |
| Modbus TCP (SMA profile, unit id 3) | `charger/sma-echarger.go` | `sma-echarger`              | eCharger, firmware 2.7+ |

Confidence markers: **verified** (tested on a device or reported by users with logs), **documented** (manual or SMA register list), **assumed** (derived, needs device verification).

## Channels and codes

| Channel                             | Modbus      | Values                                                                                                                          | Notes                                                                                                         |
| ----------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `Measurement.Operation.EVeh.ChaStt` | 35463       | 200111 not connected, 200112 sleeping, 200113 charging, 200114 charge target reached, 200106 warning, 200115 fault, 5169 locked | 200115 is reported e.g. when the cable did not lock (red LED). 200114 only seen in the eCharger register list |
| `Measurement.Operation.EVeh.Health` | 35465       | 307 ok, 455 warning, 35 alarm, 303 off, 308 on                                                                                  |                                                                                                               |
| `Parameter.Chrg.ActChaMod`          | 41471       | 4718 fast, 4719 PV surplus (optimized), 4720 target (setpoint), 4721 stop                                                       | EV Charger reports only 4718/4721 as possible values                                                          |
| `Parameter.Chrg.ChrgApv`            | 41475       | 5171 charging lock, 5172 charging release                                                                                       | "Manuelle Ladefreigabe"                                                                                       |
| `Parameter.Chrg.ChrgLok`            | 41477       | 1129 yes, 1130 no                                                                                                               | Access management, UI "Zugriffsverwaltung": "Entsperrt" (no) or "App / RFID" (yes)                            |
| `Parameter.Chrg.StpWhenFl`          | 41485       | 1129 yes, 1130 no                                                                                                               | "Trennung nach Vollladung"                                                                                    |
| `Parameter.Chrg.StpWhenFlTm`        | 41487       | minutes, 5 to 1440                                                                                                              | Standby time before disconnect after full charge                                                              |
| `Parameter.Chrg.UseEnergyMeter`     | 41489       | 1129 yes, 1130 no                                                                                                               | Operation with grid connection point meter, required for SEMP                                                 |
| `Parameter.Chrg.AMinCha`            | 41473       | 0.1 A (Modbus), 6 to 32 A                                                                                                       | Minimum charging current                                                                                      |
| `Parameter.Inverter.AcALim`         | none        | A, 6 to 32                                                                                                                      | Current limit, REST only                                                                                      |
| `Parameter.Inverter.WMaxIn`         | 44383       | W                                                                                                                               | Power limit. EV Charger range 4500 to 22000 W                                                                 |
| `Parameter.Inverter.WMax`           | 40915       | W                                                                                                                               | Maximum active power. EV Charger range 4500 to 22000 W                                                        |
| `Parameter.Inverter.WMaxInRtg`      | 33019       | W                                                                                                                               | Rated maximum input power                                                                                     |
| `Parameter.Inverter.OutPhs`         | 40183       | 326, 5194, 5195                                                                                                                 | Output phase configuration, meaning of codes unknown                                                          |
| `Parameter.PCC.ARtg`                | 40131       | A, 16 to 1000                                                                                                                   | Rated current at grid connection point                                                                        |
| `Parameter.PCC.WMaxAsym`            | none listed | W, 0 to 10000                                                                                                                   | Maximum unbalanced load, EV Charger default 4600 W                                                            |
| `Parameter.PCC.FlbInv.WMax`         | 41491       | W                                                                                                                               | Fallback power limit                                                                                          |
| `Measurement.Chrg.ModSw`            | none        | 4950 smart, 4718 boost                                                                                                          | EV Charger rotary switch position                                                                             |

## Charging release and access management

- **verified**: on the EV Charger (`EVC22-3AC-10`, 1.2.23.R) `ChrgApv` 5171 does not block charging while access management is off. A SEMP parameter dump shows `ChrgLok` 1130, `ChrgApv` 5171, mode 4718 and status 200113 at 3608 W (evcc-io/evcc#25778). evcc-io/evcc#33350 reports no difference between 5171 and 5172 on the same model.
- **verified**: with access management on (`ChrgLok` 1129), writing `ChrgApv` 5171 stops charging immediately on an eCharger 2.7.31.R, even while SEMP commands are ignored (evcc-io/evcc#36465, written via Home Assistant).
- **verified**: with access management on, the charger sets `ChrgApv` 5172 itself after app or RFID release and starts charging immediately, before any energy manager command (evcc-io/evcc#36465).
- **documented**: with access management on, the release applies to the active charging process. When charging is complete the station locks again (manual 4.5.12, 8.13.3.1).
- **documented**: after plug-in with access management on, the yellow LED shows "locked, release required" (manual 4.6, 8.13.4). Before plug-in the green LED is blinking. Status 5169 therefore implies a connected vehicle.
- **assumed**: the eCharger behaves the same with access management off. evcc-io/evcc#23789 controlled an eCharger via mode while the REST driver wrote `ChrgApv` 5171 on startup.
- **assumed**: with access management on, withdrawing the release ends the charging process on the charger rather than pausing it, since the release applies to the active process. This would reset the session energy.

App or RFID release enables charging on its own and conflicts with energy manager control. Access management must be set to "Entsperrt" for evcc. Charging is therefore controlled via the charging mode, `ChrgApv` is left untouched.

## Control paths

- **REST**: mode 4718/4721 for enable, `Inverter.AcALim` for current. Only 4718 counts as enabled. The EV Charger rotary switch must be on boost, `Measurement.Chrg.ModSw` is only meaningful there, the eCharger reports it without a physical switch. Elevated rights ("Administrator" or installer) are required for limits. Home Assistant (`alengwenus/ha-sma-ev-charger`) exposes the same channels and confirms REST works on the eCharger. Toggling between boost and stop was confirmed working on an EV Charger (`alengwenus/ha-sma-ev-charger#1`).
- **SEMP**: `On` plus `RecommendedPowerConsumption`, sent at least once per minute. eCharger firmware 2.7.31.R accepts but ignores SEMP commands and charges at full power (evcc-io/evcc#36465). A power cycle helps temporarily.
- **Modbus**: mode 4718/4721 for enable, only 4718 counts as enabled, `WMaxIn` for power. The charger switches 1p/3p internally based on the power limit (120 s changeover pause, manual 8.13.2).

## Charging mode

- **verified**: on the eCharger, switching the mode via REST stops and starts charging. The charger event log shows "Parameteränderung für Parameter Parameter.Chrg.ActChaMod ... von Wert Schnellladen auf Ladestopp" followed by "Ladestation wechselt in Ladebetrieb Ladestopp" (evcc-io/evcc#16071).
- **verified**: the eCharger falls back to its default mode "Laden mit PV-Überschuss" (4719) on its own, also while controlled by evcc (evcc-io/evcc#16071). The default mode is reportedly configurable on the charger. Treating only 4718 as enabled lets evcc restore fast charging.
- **verified**: the eCharger reports `Measurement.Chrg.ModSw` 4950 (smart) without a physical switch, which made the rotary switch check fail on enable (evcc-io/evcc#16071, fixed by evcc-io/evcc#23789).
- **verified**: without the Sunny Home Manager, the charger does not charge in PV surplus mode (evcc-io/evcc#33350).
- **verified**: `Inverter.AcALim` keeps the last value written by evcc after the session. Charging started via the SMA app then uses that limit, even in fast charging mode (evcc-io/evcc#16071).

## Power limits

- SEMP device info of an eCharger reports `MinPowerConsumption` 1518 W (6 A at 253 V) and `MaxPowerConsumption` 11040 W (16 A, 3p at 230 V).
- **documented**: the EV Charger (`EVC22-3AC-10`, package 1.2.23.R) exposes `WMaxIn` as writable with a range of 4500 to 22000 W. The minimum is above 3p at 6 A (4140 W), so the power limit cannot cover 1p or the lowest 3p currents there. `AcALim` (6 to 32 A) can.
- **assumed**: whether `WMaxIn` limits the charging power at runtime on the EV Charger or only acts as an installation limit is unverified.
- **assumed**: if the eCharger enforces the same `WMaxIn` minimum of 4500 W, single phase charging via power limit is not possible. Check the `WMaxIn` range on an eCharger, e.g. via the `min`/`max` fields of the REST parameter response.

## Modbus specifics

- Unit id 3, big endian, 32 bit registers. NaN: S32 `0x80000000`, U32 `0xFFFFFFFF`, U64 all ones, status values use the lower 24 bits (`0xFFFFFD`). STR32 strings are 16 registers.
- Writable parameters are stored in flash. SMA warns that cyclic changes destroy the flash, except grid service parameters marked in the device's "Parameter und Messwerte" document. Whether `WMaxIn` and `ChrgApv` may be written cyclically is unknown.

## Sources

- eCharger operating manual EVC22-3AC-20-BE (en-10)
- SMA Modbus interface ennexOS TI (de-12)
- eCharger Modbus register list (SW 2.7, internal)
- `alengwenus/ha-sma-ev-charger` and `alengwenus/pysmaev` (parameter fixture of an EV Charger 22 `EVC22-3AC-10`, package 1.2.23.R, charge controller SA-01.03.27)
- evcc-io/evcc#36465
