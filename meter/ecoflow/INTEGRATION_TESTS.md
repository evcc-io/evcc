# EcoFlow Integration Tests

Diese Tests prüfen die Kommunikation mit der echten EcoFlow API.

## Getestete Funktionen

| Test | Funktion | Stream | PowerStream |
|------|----------|--------|-------------|
| `ReadBatterySOC` | Ladestand (SOC) | ✅ | ✅ |
| `ReadChargingPower` | Ladegeschwindigkeit | ✅ | ✅ |
| `ReadDischargePower` | Entladegeschwindigkeit | ✅ | ✅ |
| `ControlRelay` | Starten/Stoppen Ladung/Entladung | ✅ | ❌ |
| `ControlDischarging` | Entladung steuern | ✅ (Relay) | ✅ (permanentWatts) |
| `ControlChargingSpeed` | Ladegeschwindigkeit setzen | ❌ (PV-abhängig) | ✅ |
| `FullStatus` | Komplettstatus | ✅ | ✅ |

## Voraussetzungen

```bash
# EcoFlow Developer Account mit API Keys
# https://developer-eu.ecoflow.com/us

export ECOFLOW_SN='DEINE_SERIENNUMMER'
export ECOFLOW_ACCESS_KEY='DEIN_ACCESS_KEY'
export ECOFLOW_SECRET_KEY='DEIN_SECRET_KEY'

# Optional
export ECOFLOW_DEVICE='stream'        # oder 'powerstream'
export ECOFLOW_URI='https://api-e.ecoflow.com'  # API Endpoint
export ECOFLOW_ALLOW_CONTROL='true'   # Für Steuerungs-Tests
```

## Tests ausführen

### Schnellstart: Status prüfen

```bash
cd /Users/ingmar/repos/evcc-dev/meter/ecoflow

# Status-Report (read-only, sicher)
./run_integration_test.sh --status
```

### Alle Read-Tests

```bash
./run_integration_test.sh --read
```

### Control-Tests (⚠️ Ändert Gerätezustand!)

```bash
export ECOFLOW_ALLOW_CONTROL='true'
./run_integration_test.sh --control
```

### Direkt mit Go

```bash
cd /Users/ingmar/repos/evcc-dev

# Alle Integration-Tests
go test -tags=integration -v ./meter/ecoflow/...

# Nur bestimmte Tests
go test -tags=integration -v -run TestIntegration_FullStatus ./meter/ecoflow/...
go test -tags=integration -v -run TestIntegration_Read ./meter/ecoflow/...
```

## Test-Ausgabe (Beispiel)

```
=== RUN   TestIntegration_FullStatus
    EcoFlow Device Status Report
    Device: HJ31ZD1AZH5G0342
    Type: Stream
    
    📊 BATTERY STATUS
       Ladestand (SOC): 85.0%
       Ladegeschwindigkeit: 1200 W
       Entladegeschwindigkeit: 0 W
    
    ⚡ POWER FLOW
       PV: 1500 W
       Grid: 300 W
       Load: 600 W
    
    🔌 RELAY STATUS
       AC1 (Ladung): AN ✅
       AC2 (Entladung): AUS ❌
    
    ✅ Status report completed
--- PASS: TestIntegration_FullStatus (0.45s)
```

## API-Endpunkte

### Lesen (GET)
- `/iot-open/sign/device/quota/all?sn={SN}` - Alle Gerätedaten

### Steuern (PUT)
- `/iot-open/sign/device/quota` - Relay/Einstellungen ändern

### Stream-Steuerung

```json
// Relay AC1 ein/ausschalten
{
  "sn": "SERIAL",
  "params": {
    "relay2Onoff": true  // AC1
  }
}

// Relay AC2 ein/ausschalten
{
  "sn": "SERIAL", 
  "params": {
    "relay3Onoff": true  // AC2
  }
}
```

### PowerStream-Steuerung

```json
// Entladeleistung setzen (0-600W)
{
  "sn": "SERIAL",
  "cmdCode": "WN511_SET_PERMANENT_WATTS_PACK",
  "params": {
    "permanentWatts": 1000  // Wert * 10
  }
}
```

## Hinweise

### Stream-Geräte
- **Ladung starten**: Grid-Verbindung + Relay AN
- **Ladung stoppen**: Relay AUS oder Grid trennen
- **Entladung starten**: Last anschließen + Relay AN
- **Entladung stoppen**: Relay AUS
- Ladegeschwindigkeit abhängig von PV + Grid, nicht direkt steuerbar

### PowerStream-Geräte
- **Ladung**: Automatisch über PV, SOC-Limits einstellbar
- **Entladung**: Über `permanentWatts` (0-600W) steuerbar
- **Ladegeschwindigkeit**: Abhängig von PV-Ertrag

## Troubleshooting

### API Error: "1001" / "device offline"
- Gerät ist offline oder nicht mit Cloud verbunden
- Prüfe WLAN-Verbindung des Geräts

### API Error: "1002" / "invalid credentials"
- Access/Secret Key falsch
- Keys im Developer Portal neu generieren

### Timeout
- API-Server langsam
- Netzwerkproblem
- Cache erhöhen: `cache: 30s`
