# EcoFlow Integration für evcc

Diese Dokumentation beschreibt die Integration von EcoFlow Stream und PowerStream Geräten in evcc.

## Voraussetzungen

1. **EcoFlow Developer Account**
   - Registrieren: https://developer-eu.ecoflow.com/us
   - Access Key und Secret Key erstellen

2. **Unterstützte Geräte**
   - EcoFlow STREAM Ultra
   - EcoFlow STREAM Pro
   - EcoFlow STREAM AC Pro
   - EcoFlow PowerStream (WN511, WN512)

## Konfiguration

### Als Meter (nur Anzeige)

Für Anzeige von PV-Ertrag, Grid-Bezug oder Batterie-Status:

```yaml
meters:
  - name: ecoflow_pv
    type: template
    template: ecoflow-stream
    usage: pv
    sn: DEINE_SERIENNUMMER
    accesskey: DEIN_ACCESS_KEY
    secretkey: DEIN_SECRET_KEY

  - name: ecoflow_grid
    type: template
    template: ecoflow-stream
    usage: grid
    sn: DEINE_SERIENNUMMER
    accesskey: DEIN_ACCESS_KEY
    secretkey: DEIN_SECRET_KEY

  - name: ecoflow_battery
    type: template
    template: ecoflow-stream
    usage: battery
    sn: DEINE_SERIENNUMMER
    accesskey: DEIN_ACCESS_KEY
    secretkey: DEIN_SECRET_KEY
```

### Als Site Battery

Für die Anzeige der Batterie im evcc Dashboard:

```yaml
site:
  title: Mein Zuhause
  meters:
    grid: netz
    pv:
      - ecoflow_pv
    battery:
      - ecoflow_battery
```

### Als Charger (mit Steuerung)

Für aktive Steuerung der Ladung/Entladung über Relays:

```yaml
chargers:
  - name: ecoflow_charger
    type: template
    template: ecoflow-stream
    sn: DEINE_SERIENNUMMER
    accesskey: DEIN_ACCESS_KEY
    secretkey: DEIN_SECRET_KEY
    relay: 1  # 1=AC1, 2=AC2
```

### Als Loadpoint

Kombination von Charger mit Fahrzeug-Ladung:

```yaml
loadpoints:
  - title: EcoFlow Speicher
    charger: ecoflow_charger
    mode: pv
    phases: 1
    mincurrent: 6
    maxcurrent: 16
```

## Vollständiges Beispiel

```yaml
network:
  schema: http
  host: evcc.local
  port: 7070

meters:
  # Haupt-Netzmessung (z.B. Shelly 3EM)
  - name: netz
    type: shelly
    uri: 192.168.1.100

  # EcoFlow Stream als PV-Meter
  - name: ecoflow_pv
    type: template
    template: ecoflow-stream
    usage: pv
    sn: BK61ZE1B2H6H0912
    accesskey: Ms0Nefw3xBOHZMA36l8fD7IzXteWLvLL
    secretkey: uzDj9L9F5v5DFGObypJH5vlAcHkNPYn8

  # EcoFlow Stream als Batterie-Meter
  - name: ecoflow_battery
    type: template
    template: ecoflow-stream
    usage: battery
    sn: BK61ZE1B2H6H0912
    accesskey: Ms0Nefw3xBOHZMA36l8fD7IzXteWLvLL
    secretkey: uzDj9L9F5v5DFGObypJH5vlAcHkNPYn8

chargers:
  # EcoFlow Stream als steuerbarer Charger
  - name: ecoflow_charger
    type: template
    template: ecoflow-stream
    sn: BK61ZE1B2H6H0912
    accesskey: Ms0Nefw3xBOHZMA36l8fD7IzXteWLvLL
    secretkey: uzDj9L9F5v5DFGObypJH5vlAcHkNPYn8
    relay: 1

site:
  title: Mein Zuhause
  meters:
    grid: netz
    pv:
      - ecoflow_pv
    battery:
      - ecoflow_battery

loadpoints:
  - title: EcoFlow Speicher
    charger: ecoflow_charger
    mode: pv
```

## Funktionen

### Stream Geräte

| Funktion | Unterstützt | Beschreibung |
|----------|-------------|--------------|
| PV-Leistung | ✅ | Echtzeit PV-Ertrag |
| Grid-Leistung | ✅ | Netzeinspeisung/-bezug |
| Batterie-Leistung | ✅ | Lade-/Entladeleistung |
| Batterie-SOC | ✅ | Ladestand in % |
| Relay-Steuerung | ✅ | AC1/AC2 ein/aus |
| Ladegeschwindigkeit | ❌ | Abhängig von PV/Grid |

### PowerStream Geräte

| Funktion | Unterstützt | Beschreibung |
|----------|-------------|--------------|
| PV-Leistung | ✅ | PV1 + PV2 Summe |
| Grid-Leistung | ✅ | Inverter-Ausgang |
| Batterie-Leistung | ✅ | Lade-/Entladeleistung |
| Batterie-SOC | ✅ | Ladestand in % |
| Entladeleistung | ✅ | permanentWatts (0-600W) |

## Troubleshooting

### "missing sn, accessKey or secretKey"
- Prüfe ob alle drei Parameter in der Config gesetzt sind
- Keys aus dem Developer Portal kopieren

### "mqtt certification failed"
- API-Zugang prüfen
- Keys eventuell neu generieren

### "device offline"
- Gerät mit WLAN verbunden?
- EcoFlow Cloud erreichbar?

### Verzögerte Werte
- MQTT liefert Live-Daten (1-2s Latenz)
- Bei MQTT-Problemen: REST API Fallback (10s Cache)

## API-Dokumentation

- [EcoFlow Developer Portal](https://developer-eu.ecoflow.com/us)
- [Stream API Docs](https://developer-eu.ecoflow.com/us/document/bkw)
- [PowerStream API Docs](https://developer-eu.ecoflow.com/us/document/wn511)
