# EcoFlow Integration für evcc

Diese Dokumentation beschreibt die Integration von EcoFlow Stream- und PowerStream-Geräten in evcc.

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

### Als Meter (PV, Grid, Batterie)

```yaml
meters:
  # PV-Ertrag
  - name: ecoflow_pv
    type: template
    template: ecoflow-stream
    usage: pv
    sn: DEINE_SERIENNUMMER
    accesskey: DEIN_ACCESS_KEY
    secretkey: DEIN_SECRET_KEY

  # Grid-Bezug/Einspeisung
  - name: ecoflow_grid
    type: template
    template: ecoflow-stream
    usage: grid
    sn: DEINE_SERIENNUMMER
    accesskey: DEIN_ACCESS_KEY
    secretkey: DEIN_SECRET_KEY

  # Batterie (mit automatischer Steuerung!)
  - name: ecoflow_battery
    type: template
    template: ecoflow-stream
    usage: battery
    sn: DEINE_SERIENNUMMER
    accesskey: DEIN_ACCESS_KEY
    secretkey: DEIN_SECRET_KEY
```

### Site-Konfiguration

```yaml
site:
  title: Mein Zuhause
  meters:
    grid: netz          # Haupt-Netzmessung
    pv:
      - ecoflow_pv
    battery:
      - ecoflow_battery  # Mit BatteryController-Support!
```

## Automatische Batterie-Steuerung

Bei `usage: battery` wird automatisch:

1. **MQTT-Verbindung** für Live-Updates und Steuerung aufgebaut
2. **`api.BatteryController`** implementiert für evcc's Batterie-Features:

| evcc Feature | EcoFlow Aktion |
|--------------|----------------|
| **BatteryHold** (Entlade-Sperre) | Relays aus → Batterie entlädt nicht |
| **BatteryNormal** | Relays an → Normalbetrieb |
| **BatteryCharge** | Relays an (Grid-Laden nicht direkt unterstützt) |

### Unterstützte evcc-Features

- ✅ **Prioritäts-SOC**: Batterie zuerst laden bis X%
- ✅ **Batterie-unterstütztes Laden**: Batterie für Fahrzeug freigeben
- ✅ **Entlade-Sperre**: Bei Schnellladen/Planer Batterie schonen
- ⚠️ **Netzladen**: Nicht direkt unterstützt (EcoFlow-Limitation)

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

  # EcoFlow Stream - PV
  - name: ecoflow_pv
    type: template
    template: ecoflow-stream
    usage: pv
    sn: BK61ZE1B2H6H0912
    accesskey: DEIN_ACCESS_KEY
    secretkey: DEIN_SECRET_KEY

  # EcoFlow Stream - Batterie (mit Steuerung)
  - name: ecoflow_battery
    type: template
    template: ecoflow-stream
    usage: battery
    sn: BK61ZE1B2H6H0912
    accesskey: DEIN_ACCESS_KEY
    secretkey: DEIN_SECRET_KEY

site:
  title: Mein Zuhause
  meters:
    grid: netz
    pv:
      - ecoflow_pv
    battery:
      - ecoflow_battery

  # Batterie-Priorität konfigurieren
  prioritySoc: 50        # Batterie bis 50% mit Priorität laden
  bufferSoc: 80          # Ab 80% für Fahrzeug freigeben
  bufferStartSoc: 90     # Batterie-Boost ab 90%

# Wallbox für EV
chargers:
  - name: wallbox
    type: template
    template: go-e
    host: 192.168.1.101

loadpoints:
  - title: Garage
    charger: wallbox
    mode: pv
```

## Technische Details

### MQTT-Kommunikation

Die Batterie-Integration nutzt EcoFlows offizielle MQTT-API:

| Topic | Richtung | Beschreibung |
|-------|----------|--------------|
| `/open/{account}/{sn}/quota` | Subscribe | Live-Daten (1-2s Latenz) |
| `/open/{account}/{sn}/set` | Publish | Relay-Steuerung |

### Relay-Mapping

| Relay | EcoFlow Name | Funktion |
|-------|--------------|----------|
| AC1 | `relay2Onoff` | Haupt-AC-Ausgang |
| AC2 | `relay3Onoff` | Sekundär-AC-Ausgang |

### Daten-Quellen

1. **MQTT** (bevorzugt): Live-Updates alle 1-2 Sekunden
2. **REST API** (Fallback): Polling alle 10 Sekunden

## Troubleshooting

### "mqtt not available for battery control"
- MQTT-Verbindung fehlgeschlagen
- Prüfe Netzwerk zu `mqtt-e.ecoflow.com:8883`
- API-Keys korrekt?

### Batterie-Steuerung reagiert nicht
- Prüfe ob `usage: battery` gesetzt ist
- Nur Battery-Meter bekommen MQTT-Steuerung

### Verzögerte Werte
- MQTT nicht verbunden → REST-Fallback aktiv
- Prüfe evcc-Logs auf MQTT-Fehler

## API-Dokumentation

- [EcoFlow Developer Portal](https://developer-eu.ecoflow.com/us)
- [Stream API Docs](https://developer-eu.ecoflow.com/us/document/bkw)
- [evcc Battery Features](https://docs.evcc.io/en/docs/features/battery)
