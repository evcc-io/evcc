# EVCC Installationsassistent

Beantworte die folgenden Fragen um die EVCC Konfiguration zu erstellen.

## Allgemein

### Verwendest Du MQTT und sollen Daten aus MQTT verarbeitet werden?

[JA: MQTT konfigurieren](#mqtt-konfigurieren)
[NEIN: Weiter](#möchtest-du-messwerte-in-influxdb-erfassen-und-mit-grafana-visualisieren)

### Möchtest Du Messwerte in InfluxDB erfassen und mit Grafana visualisieren?

[JA: InfluxDB konfigurieren](#influxdb-konfigurieren)
[NEIN: Weiter](#wallbox)

## Wallbox

Beantworte die folgenden Fragen um die Wallbox zu konfigurieren. Falls noch keine Wallbox vorhanden ist, kann EVCC dennoch getestet werden. Siehe [EVCC ohne Wallbox ausprobieren](#evcc-ohne-wallbox-ausprobieren).

Welche Wallbox soll verwendet werden:
- Wallbe
- KEBA Connect
- go-eCharger
- andere Wallbox mit Phoenix Controller

## Zähler konfigurieren

ping

## Fahrzeug konfigurieren

pong



# EVCC ohne Wallbox ausprobieren

Wenn [Zähler](#zähler-prüfen) oder [Fahrzeug](#fahrzeug-prüfen) konfiguriert sind, kann die Konfiguration jeweils überprüft werden.

## Zähler prüfen

    evcc -c evcc.yaml meter

[Zurück zur Zählerkonfiguration](#zähler-konfigurieren)

## Fahrzeug prüfen

    evcc -c evcc.yaml vehicle

[Zurück zur Fahrzeugkonfiguration](#fahrzeug-konfigurieren)



# Anweisungen

## Allgemein

### MQTT konfigurieren

[Zurück](#verwendest-du-mqtt-und-sollen-daten-aus-mqtt-verarbeitet-werden)
[Weiter](#möchtest-du-messwerte-in-influxdb-erfassen-und-mit-grafana-visualisieren)

### InfluxDB konfigurieren

foo

## Wallbox konfigurieren

bar
