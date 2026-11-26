# SMA EV Charger and eCharger

SMA ships two generations of AC chargers that share one parameter model:

- **EV Charger** (`EVC7.4-1AC-10`, `EVC22-3AC-10`), firmware 1.2.x, hardware rotary switch (smart/boost).
- **eCharger** (`EVC22-3AC-20`), firmware 2.x, ennexOS based, no rotary switch.

All protocols expose the same SMA channels (`Parameter.*`, `Measurement.*`) with the same enum codes. Shared channel ids and codes live in `charger/sma`; protocol drivers only implement transport and control.

| Protocol                            | Driver                  | Template                                     | Hardware                |
| ----------------------------------- | ----------------------- | -------------------------------------------- | ----------------------- |
| REST (`/api/v1`, OAuth token)       | `charger/sma-rest.go`   | `smaevcharger` (deprecated, type `sma-rest`) | EV Charger, eCharger    |
| SEMP (`/SEMP`, XML)                 | `charger/semp.go`       | `semp-sma`                                   | EV Charger, eCharger    |
| Modbus TCP (SMA profile, unit id 3) | `charger/sma-modbus.go` | `smaecharger` (type `sma-modbus`)            | eCharger, firmware 2.7+ |

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
| `Parameter.Chrg.MinSwTm`            | none        | 6715 = 120 s                                                                                                                    | Minimum relay switching time, "Minimale Schaltdauer Relais", default 120 s                                    |
| `Parameter.Chrg.UseEnergyMeter`     | 41489       | 1129 yes, 1130 no                                                                                                               | Operation with grid connection point meter, required for SEMP                                                 |
| `Parameter.Chrg.AMinCha`            | 41473       | 0.1 A (Modbus), 6 to 32 A                                                                                                       | Minimum charging current                                                                                      |
| `Parameter.Inverter.AcALim`         | none        | A, 6 to 32                                                                                                                      | Current limit, REST only                                                                                      |
| `Parameter.Inverter.WMaxIn`         | 44383       | W                                                                                                                               | Power limit. EV Charger range 4500 to 22000 W                                                                 |
| `Parameter.Inverter.WMax`           | 40915       | W                                                                                                                               | Maximum active power. EV Charger range 4500 to 22000 W                                                        |
| `Parameter.Inverter.WMaxInRtg`      | 33019       | W                                                                                                                               | Rated maximum input power                                                                                     |
| `Parameter.Inverter.WMinIn`         | 44387       | W                                                                                                                               | Minimum input power, writable, not seen in REST dumps                                                         |
| `Parameter.Inverter.OutPhs`         | 40183       | 326, 5194, 5195                                                                                                                 | Output phase configuration, meaning of codes unknown                                                          |
| `Parameter.PCC.ARtg`                | 40131       | A, 16 to 1000                                                                                                                   | Rated current at grid connection point                                                                        |
| `Parameter.PCC.WMaxAsym`            | none listed | W, 0 to 10000                                                                                                                   | Maximum unbalanced load, EV Charger default 4600 W                                                            |
| `Parameter.PCC.FlbInv.WMax`         | 41491       | W                                                                                                                               | Fallback power limit                                                                                          |
| `Measurement.Chrg.ModSw`            | none        | 4950 smart, 4718 boost                                                                                                          | EV Charger rotary switch position                                                                             |
| `Parameter.Chrg.CblLok`             | 41499       | 303 off, 308 on                                                                                                                 | Cable lock                                                                                                    |
| `Measurement.Acs.AuthRegistStt`     | none        | 302 none                                                                                                                        | RFID card registration status                                                                                 |

## Charging release and access management

- **verified**: on the EV Charger (`EVC22-3AC-10`, 1.2.23.R) `ChrgApv` 5171 does not block charging while access management is off. A SEMP parameter dump shows `ChrgLok` 1130, `ChrgApv` 5171, mode 4718 and status 200113 at 3608 W (evcc-io/evcc#25778). evcc-io/evcc#33350 reports no difference between 5171 and 5172 on the same model.
- **verified**: with access management on (`ChrgLok` 1129), writing `ChrgApv` 5171 stops charging immediately on an eCharger 2.7.31.R, even while SEMP commands are ignored (evcc-io/evcc#36465, written via Home Assistant).
- **verified**: with access management on, the charger sets `ChrgApv` 5172 itself after app or RFID release and starts charging immediately, before any energy manager command (evcc-io/evcc#36465).
- **documented**: with access management on, the release applies to the active charging process. When charging is complete the station locks again (manual 4.5.12, 8.13.3.1).
- **documented**: after plug-in with access management on, the yellow LED shows "locked, release required" (manual 4.6, 8.13.4). Before plug-in the green LED is blinking. Status 5169 therefore implies a connected vehicle.
- **assumed**: the eCharger behaves the same with access management off. evcc-io/evcc#23789 controlled an eCharger via mode while the REST driver wrote `ChrgApv` 5171 on startup.
- **assumed**: with access management on, withdrawing the release ends the charging process on the charger rather than pausing it, since the release applies to the active process. This would reset the session energy.

evcc controls charging via the charging mode and leaves `ChrgApv` untouched. Access management "App / RFID" is allowed: until the release is given (`ChrgLok` 1129 and `ChrgApv` not 5172) the drivers report waiting for authorization. The SEMP parameters endpoint exposes both channels as well (evcc-io/evcc#25778).

## Control paths

- **REST**: `WSptMax` and `WSptMin` set to the same negative value for power, 0 to disable, rewritten at half the control timeout. Without writable setpoint channels (EV Charger, unverified) mode 4718/4721 for enable and `Inverter.WMaxIn` for power, clamped to the `min`/`max` reported for the parameter. With a 4500 W minimum, as on the EV Charger, 1p charging and phase switching are not possible there.
- **SEMP**: `On` plus `RecommendedPowerConsumption`, sent at least once per minute. eCharger firmware 2.7.31.R accepts but ignores SEMP commands and charges at full power (evcc-io/evcc#36465). A power cycle helps temporarily.
- **Modbus**: `WSptMax` (41431) and `WSptMin` (41433) set to the same negative value for power, 0 to disable, rewritten at half the control timeout (41525). Enabled while `WSptMin` is not 0. The charger switches 1p/3p internally based on the power (120 s changeover pause, manual 8.13.2).

## Charging mode

- **verified**: on the eCharger, switching the mode via REST stops and starts charging. The charger event log shows "Parameteränderung für Parameter Parameter.Chrg.ActChaMod ... von Wert Schnellladen auf Ladestopp" followed by "Ladestation wechselt in Ladebetrieb Ladestopp" (evcc-io/evcc#16071).
- **verified**: the eCharger falls back to its default mode "Laden mit PV-Überschuss" (4719) on its own, also while controlled by evcc (evcc-io/evcc#16071). The default mode is reportedly configurable on the charger. Treating only 4718 as enabled lets evcc restore fast charging.
- **verified**: the eCharger reports `Measurement.Chrg.ModSw` 4950 (smart) without a physical switch, which made the rotary switch check fail on enable (evcc-io/evcc#16071, fixed by evcc-io/evcc#23789).
- **verified**: on the EV Charger the rotary switch limits the possible values of `ActChaMod` (REST `possibleValues`): boost position allows 4718 and 4721, smart position allows 4719, 4720 and 4721 (three parameter dumps of `EVC22-3AC-10`, 1.2.23.R: `alengwenus/ha-sma-ev-charger`, `J0B10/MAD-MyEnergy`, `PaulV71/gwHEMS-Userbase`). On the eCharger 4718 is writable while `ModSw` reports 4950 (evcc-io/evcc#16071).
- **verified**: without the Sunny Home Manager, the charger does not charge in PV surplus mode (evcc-io/evcc#33350).
- **verified**: `Inverter.AcALim` keeps the last value written by evcc after the session. Charging started via the SMA app then uses that limit, even in fast charging mode (evcc-io/evcc#16071).

## Startup checks

REST fails on startup when neither the setpoint channels nor `ActChaMod` and `WMaxIn` are `editable` for the configured account. Writing limits requires an administrator or installer account, a plain user account can read values (`alengwenus/ha-sma-ev-charger#38`, `SpongeTroN86/ioBroker.sma-echarger`).

All drivers only read on startup and warn on settings that conflict with evcc control: EV Charger rotary switch in smart position (REST without setpoint), "Trennung nach Vollladung" (`StpWhenFl`) enabled (all, SEMP only when the parameters endpoint provides it). SEMP additionally warns when "Betrieb mit Netzanschlusspunktzähler" (`UseEnergyMeter`) is disabled, again only when the parameters endpoint provides it.

- **documented**: SEMP control requires `UseEnergyMeter` enabled and a meter configured on the charger (`semp-sma` setup). REST and Modbus control do not depend on it.
- **verified**: the EV Charger answers SEMP with `EMSignalsAccepted` true even with `UseEnergyMeter` 1130, so the SEMP status does not reveal a missing meter (evcc-io/evcc#33350).
- **unknown**: `UseEnergyMeter` is missing from the EV Charger `/SEMP/Parameters` output. Whether a configured meter actually communicates is not exposed by any known channel.

## SEMP parameters endpoint

`/SEMP/Parameters` is read only and complements the SEMP device document.

- **verified**: the EV Charger (`EVC22-3AC-10`, 1.2.23.R) exposes `Measurement.Operation.EVeh.ChaStt`, `Measurement.Operation.EVeh.Health`, `Measurement.Operation.Health`, `Measurement.Operation.Evt.Msg`, `Measurement.ChaSess.WhIn`, `Measurement.Metering.GridMs.TotWIn.ChaSta`, `Measurement.Chrg.ModSw`, `Parameter.Chrg.ActChaMod`, `Parameter.Chrg.ChrgApv`, `Parameter.Chrg.ChrgLok`, `Parameter.Chrg.Plan.En`, `Parameter.Chrg.Plan.StopTm`, `Parameter.Chrg.StpWhenFl`, `Parameter.Chrg.StpWhenFlTm`, `Parameter.Inverter.OutPhs`, `Parameter.Nameplate.PkgRev` and `Setpoint.PlantControl.PCC.ChrgActCnt` (identical in evcc-io/evcc#13695 and evcc-io/evcc#25778).
- Not exposed: limits (`AcALim`, `WMaxIn`, `AMinCha`), total energy and per phase values.
- **verified**: the eCharger (`EVC22-3AC-20`, 2.7.31.R) exposes the endpoint with the same namespace and the same status, mode and release channels: `EVeh.ChaStt`, `EVeh.Health`, `Operation.Health`, `Operation.Evt.Msg`, `ChaSess.WhIn`, `Metering.GridMs.TotWIn.ChaSta`, `Chrg.ActChaMod`, `Chrg.ChrgApv`, `Chrg.ChrgLok`, `Chrg.Plan.En`, `Chrg.Plan.StopTm`, `Chrg.StpWhenFl`, `Chrg.StpWhenFlTm`, `Inverter.OutPhs`, `Nameplate.PkgRev`, `Setpoint.PlantControl.PCC.ChrgActCnt` (evcc-io/evcc#36465).
- **verified**: additionally on the eCharger: `Parameter.Chrg.CblLok` (cable lock, 303 off), `Measurement.Acs.AuthRegistStt` (RFID card registration, 302), `Parameter.Sys.DevSigBri`, `Parameter.Sys.AmbLghtBri`, `Parameter.Sys.UseAmbLght`, `Parameter.Nameplate.Location` and `Setpoint.PlantControl.Inverter.WModCfg.WCtlComCfg.WSptMaxNom` / `WSptMinNom` (0 and -100, meaning unknown, possibly the power setpoint range used by SEMP).
- **verified**: the eCharger does not expose `Measurement.Chrg.ModSw` here. Neither generation exposes `Parameter.Chrg.UseEnergyMeter`, so the SEMP meter check never triggers with the known firmware.
- **assumed**: the charger resets `ChrgApv` to 5171 itself when a charging session ends, also with access management off. In the eCharger dump `ChrgApv` and `ChaSess.WhIn` (reset to 0) share the same timestamp to the millisecond, while `ChrgLok` stayed 1130 since the last restart. This matches `ChrgApv` having no effect during free charging.

## Power limits

- SEMP device info reports `MinPowerConsumption` 1518 W on the EV Charger `EVC22-3AC-10` 1.2.23.R (max 22000 W in evcc-io/evcc#25778, max 7400 W in evcc-io/evcc#33350) and on the eCharger 2.7.31.R (max 11040 W, evcc-io/evcc#36465), and 1524 W on the eCharger 2.7.22.R (max 11000 W, evcc-io/evcc discussion 33281). How the minimum is derived is unknown (1518 W matches both 6 A at 253 V and 6.6 A at 230 V). It is well below the `WMaxIn` minimum, so SEMP power control reaches single phase charging while `WMaxIn` does not.
- **assumed**: the Modbus pair 44383 `Inverter.WMaxIn` / 44387 `Inverter.WMinIn` corresponds to SEMP `MaxPowerConsumption` / `MinPowerConsumption`. `WMinIn` appears in none of the REST parameter dumps and is writable, so it may be a setting rather than the value SEMP reports. The Modbus driver already reports 44387 as minimum power (`GetMinMaxPower`) and shows it in the diagnosis as "Min power".
- **documented**: `WMaxIn` reports a range of 4500 to 22000 W on the EV Charger (three parameter dumps). The eCharger ioBroker adapter (`SpongeTroN86/ioBroker.sma-echarger`) uses 4500 to 11000 W. The minimum comes from the parameter metadata of the charger, SMA gives no rationale. `WMax` has the same range and FHEM labels `WMaxIn` "Nennwirkleistung", which suggests a rating or installation limit rather than a control setpoint.
- **documented**: `erwin1/homecontrol` controls an SMA charger via REST with mode 4718/4721 and `WMaxIn` between 1380 and 7400 W, written together with mode 4718. The model is not stated, the 7400 W cap suggests an `EVC7.4-1AC-10`.
- **documented**: `AcALim` (6 to 32 A, fractional values accepted) is available via REST on both generations. It is not part of the Modbus profile.
- **documented**: the EV Charger (`EVC22-3AC-10`, package 1.2.23.R) exposes `WMaxIn` as writable with a range of 4500 to 22000 W. The minimum is above 3p at 6 A (4140 W), so the power limit cannot cover 1p or the lowest 3p currents there. `AcALim` (6 to 32 A) can.
- **assumed**: whether `WMaxIn` limits the charging power at runtime on the EV Charger or only acts as an installation limit is unverified.
- **verified** (eCharger `EVC22-3AC-20`, FW 2.7.31.R, Modbus): 44387 `WMinIn` reads -1380 W and 33019 `WMaxInRtg` -22080 W. The 4500 W minimum of the EV Charger does not apply, 1p charging via power limit works.
- **verified** (same device): 44383 `WMaxIn` only accepts negative values (consumption), positive values fail with Modbus exception 3. Whether REST on the eCharger uses the same sign is unknown.
- **verified** (same device): the charger derives the phase current from the power limit and the measured voltage and delivers about 0.5 A less. Below 6 A per phase at 3p it pauses and continues on 1p, e.g. held 3p at 4545 W but dropped to 1p at 4250 W and 4140 W with 234 to 239 V.
- **verified** (same device): 4223 W at 3p, exactly 6.00 A at the highest measured phase voltage, also dropped to 1p. On 1p the current follows the setpoint without the 0.5 A offset (-1393 W, 5.9 A).
- **assumed**: the charger switches to 1p below 4500 W, the `WMaxIn` minimum of the EV Charger and of the eCharger ioBroker adapter. 4500 W at 230 V is 6.52 A per phase and fits all data points. Whether the threshold depends on the voltage or has a hysteresis from 1p to 3p is unknown. The Modbus driver keeps 3p setpoints at 4500 W or more.
- **verified** (same device): each start and each 1p/3p switch pauses about 2 to 2.5 min, caused by `Chrg.MinSwTm` (minimum relay switching time, default 120 s). REST warns on startup at 120 s, longer than the loadpoint phase switch grace period of 60 s.
- **verified** (eCharger, FW 2.7.31.R): the web UI parameter export names the channel `Parameter.Chrg.MinSwTm`. It is an enum with 15, 30, 60, 90 or 120 s, raw value 6715 for 120 s, codes of the other values unknown. `/SEMP/Parameters` does not contain it, nor `UseEnergyMeter`.
- **documented**: FHEM `76_SMAEVCharger.pm` lists `Parameter.Chrg.MinSwTms`, "Minimale Schaltdauer Relais", possibly the EV Charger name. It is not part of the eCharger Modbus register list.

## External active power setpoint

SMA inverters share the `Inverter.WModCfg.WCtlComCfg` channel family for active power control via communication ("Externe Wirkleistungsvorgabe"). Known from inverter parameter lists: `WSptMaxNom` (Modbus 44039, "Maximale Wirkleistung") and `WSptMinNom` (44041, "Minimale Wirkleistung") as S32 percent of nominal power with 0.01 resolution, `WSpt` (40149/44425, "Wirkleistungsvorgabe" in W), `WCtlComCfg.WMaxIn` (41251, "Begrenzung der Leistungsaufnahme in W") and the fallback group `CtlComCfg.WCtlCom.CtlComMssMod` / `TmsOut` / `FlbWNom` (41193, 41195, 41197). Signs follow the generator convention, positive means output, negative means consumption.

- **assumed**: on the eCharger `Setpoint.PlantControl.Inverter.WModCfg.WCtlComCfg.WSptMaxNom` 0 and `WSptMinNom` -100 define the allowed range of an external setpoint, from no output to consuming 100 % of nominal power. They are not a minimum charging power.
- **assumed**: as `Setpoint.PlantControl` channels they are written by plant control, likely the Sunny Home Manager or the charger itself when converting SEMP commands. SMA permits cyclic writes only for such setpoints.
- **documented**: the eCharger Modbus register list contains neither 44039/44041 nor `WSpt`, only `Setpoint.PlantControl.Inverter.FstStop` (41253). Of the fallback group it lists 41193 and 41525 (`TmsOutPrc`).
- **verified** (eCharger, FW 2.7.31.R): the unlisted holding registers answer anyway. 44039 reads 0 and 44041 -10000, as FIX2 percent 0 % and -100 %, matching `WSptMaxNom` / `WSptMinNom` from REST. 44425 and 40149 (`WSpt`) read NaN, so they exist but are not set. Writing them is untested.
- **documented** (Fobbe1337 in evcc-io/evcc#35881): `WMaxIn` is the maximum power configured by the user and must be left unchanged. Control uses `Setpoint.PlantControl.Inverter.WModCfg.WCtlComCfg.WSptMax` (41431, S32 W) and `WSptMin` (41433, S32 W): set `WSptMin` or both to the same value. Alternatively `WSptMaxNom` / `WSptMinNom` (44039 / 44041, S32 0.01 %) as percent of `WMaxIn`. PV surplus charging works with these setpoints.
- **documented** (Fobbe1337): the limit applies in fast charging and PV surplus mode. In PV surplus mode the internal energy calculation respects `WSptMin`, in fast charging `WSptMax` is ignored and set to `WMaxIn`. Setting both to the same value makes the setpoint exact in all modes, so the charger does not regulate on its own.
- **verified** (eCharger, FW 2.7.31.R): `WMaxIn` reads -11000 W after a deliberate change from 22000 to 11000 W in the UI, below the rated 22080 W. A configured `WMaxIn` below the rating is normal.
- **verified** (same device): 41525 `TmsOutPrc` reads 60000, the 600 s control timeout in 0.01 s. `CtlComMssMod` (41193) reads 2507 with `FlbWMinNom` -100 %: after the timeout the charger falls back to full power. 2506 reportedly stops charging instead.
- **assumed**: the setpoints follow the generator convention, consumption negative. Charging pauses at 0 without ending the session or switching the mode (unverified). The REST channel `Parameter.Inverter.CtlComCfg.WCtlCom.TmsOutPrc` is assumed to report seconds. Whether the setpoint channels are available via REST is unknown.
- **documented**: in inverter parameter lists `Inverter.WMax` is "Nennwirkleistung WMaxOut" (menu "Maximale Wirkleistungsabgabe", configurable below the rating), `Inverter.WLim` is "Bemessungswirkleistung WMaxOutRtg", and percent values refer to `WMax` (fallback "in % von WMax", manual mode "in % Pmax", P(U) reference "Maximale Wirkleistung WMax (WMaxInOut)").
- **assumed**: the same pattern applies to the input direction of the charger. `WMaxIn` is the configured nominal input power the station is installed for and the 100 % reference of `WSptMinNom`, `WMaxInRtg` is the rating. `WMax` is a leftover of the inverter model on a charger without feed-in and mirrors `WMaxIn` in all dumps. The 4500 W minimum fits a nominal installation value.
- **assumed**, confirmed by the statement above: writing `WMaxIn` cyclically therefore changes the nominal power instead of a setpoint: the 4500 W minimum applies, flash wear is likely, and percent based setpoints from plant control shift their reference.

## Modbus specifics

- Unit id 3, big endian, 32 bit registers. NaN: S32 `0x80000000`, U32 `0xFFFFFFFF`, U64 all ones, status values use the lower 24 bits (`0xFFFFFD`). STR32 strings are 16 registers.
- Writable parameters are stored in flash. SMA warns that cyclic changes destroy the flash, except grid service parameters marked in the device's "Parameter und Messwerte" document. Whether `WMaxIn` and `ChrgApv` may be written cyclically is unknown.
- **verified** (eCharger, FW 2.7.31.R): every write to 44383 appears in the event log as a parameter change of `Inverter.WMaxIn`. This only shows that the charger logs the change, not that it is persisted to flash. No alternative setpoint register is known.
- **verified** (same device): 44383 still reads the last written value hours later. It is retained, whether in flash or RAM is unknown.

## Sources

- eCharger operating manual EVC22-3AC-20-BE (en-10)
- SMA Modbus interface ennexOS TI (de-12)
- eCharger Modbus register list (SW 2.7, internal)
- `alengwenus/ha-sma-ev-charger` and `alengwenus/pysmaev` (parameter fixture of an EV Charger 22 `EVC22-3AC-10`, package 1.2.23.R, charge controller SA-01.03.27)
- evcc-io/evcc#36465
- `SpongeTroN86/ioBroker.sma-echarger` (eCharger, REST via HTTPS)
- FHEM `76_SMAEVCharger.pm` (EV Charger, German value names)
- `erwin1/homecontrol` (REST control and `/SEMP/Parameters`)
- `J0B10/MAD-MyEnergy`, `PaulV71/gwHEMS-Userbase` (REST parameter dumps of `EVC22-3AC-10`)
- SMA "Parameter und Messwerte" STP x.0-3SE-40 and `StefanLSA/py-sma-modbus2` (inverter `WCtlComCfg` channels)
