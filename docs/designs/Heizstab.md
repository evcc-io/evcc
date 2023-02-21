# Heizstab

## Charger

- MaxCurrentEx: granulare Steuerung der maximalen Stromstärke
- Enable(on/off): nutzt MaxCurrent
- Enabled: Status "heizt gerade"- hier wäre es hilfreich zu wissen, ob der Heizstab gerade heizt oder nicht
- Soc: Speicherladezustand/ Temperatur
- CurrentPower: aktuelle Leistung gem. eingestellter Heizleistung

Randbedingung: Loadpoint muss so konfiguriert seit, dass die Stromvorgabe mit der Heizleistung übereinstimmt.

### Modbus Register

- Heizleistung: write (u)int16
- Temperatur: read int16 (ggf. mit Faktor 10 oder 100)
- Status: read int16 (0=aus, 1=an)

## UI

- Icon: Heizstab/ Warmwasser
- Keine Fahrzeugzuordnung
- Anzeige in °C statt % (Wertebereich 20-80°C)
- Kein Zielladen, kein MinSoc

## Fehlender Features

- Prioritätssteuerung: Fahrzeug vor Heizstab und umgekehrt
