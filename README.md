# evcc 🚘☀️

[![Build](https://github.com/evcc-io/evcc/actions/workflows/nightly.yml/badge.svg)](https://github.com/evcc-io/evcc/actions/workflows/nightly.yml)
[![Translation](https://hosted.weblate.org/widgets/evcc/-/evcc/svg-badge.svg)](https://hosted.weblate.org/engage/evcc/)
[![Open in Visual Studio Code](https://img.shields.io/static/v1?logo=visualstudiocode&label=&message=Open%20in%20VS%20Code&labelColor=2c2c32&color=007acc&logoColor=007acc)](https://open.vscode.dev/evcc-io/evcc)
[![OSS hosting by cloudsmith](https://img.shields.io/badge/OSS%20hosting%20by-cloudsmith-blue?logo=cloudsmith)](https://cloudsmith.io/~evcc/packages/)
[![Latest Version](https://img.shields.io/github/release/evcc-io/evcc.svg)](https://github.com/evcc-io/evcc/releases)

evcc is an extensible EV Charge Controller and home energy management system. Featured in [PV magazine](https://www.pv-magazine.de/2021/01/15/selbst-ist-der-groeoenlandhof-wallbox-ladesteuerung-selbst-gebaut/).

![Screenshot](assets/github/screenshot.webp)

## Features

- simple and clean user interface
- support for many [EV chargers](https://docs.evcc.io/en/docs/devices/chargers):
  - ABB, ABL, ABL Sursum, Alfen, alpha innotec, Alphatec, Amperfied, Ampure, Audi, AUTEL, Autoaid, Bender, BMW, Bosch, Buderus, Bösch, cFos, Charge Amps, Compleo, CTA All-In-One, CUBOS, Cupra, Dadapower, DaheimLaden, Daikin, Delta, E.ON Drive, E3/DC, Easee, Ebee, echarge, EcoHarmony, Edgetech, Elco, Elecq, eledio, Elli, EM2GO, EN+, enercab, Ensto, EntraTek, ESL, eSystems, Etrel, EVBox, Free2Move, Free2move eSolutions, Fronius, Garo, go-e, Hardy Barth, HardyBarth, Heidelberg, Hesotec, Homecharge, Huawei, IDM, Innogy, INRO, Juice, Junkers, Kathrein, KEBA, Kermi, Kontron Solar, Kostal, KSE, LadeFoxx, Lambda, LRT, Mennekes, my-PV, Nibe, Novelan, NRGkick, OBO Bettermann, OpenEVSE, openWB, Optec, Orbis, PC Electric, Peblar, Phoenix Contact, Plugchoice, Porsche, Pracht, Pulsares, Pulsatrix, Qcells, Roth, Schneider, Schrack, SENEC, Siemens, Skoda, SMA, Smartfox, SolarEdge, Solax, Sonnen, Spelsberg, Stark in Strom, Stiebel Eltron, Sungrow, Tecalor, TechniSat, Tesla, Tigo, TinkerForge, Ubitricity, V2C Trydan, Vaillant, Vestel, Victron, Viessmann, Viridian EV, Volkswagen, Volt Time, Wallbe, wallbox, Walther Werke, Webasto, Weidmüller, Wolf, Zaptec, Zewotherm, ZJ Beny. [Read more.](https://docs.evcc.io/en/docs/devices/chargers)
  - **EEBus** support (Elli, PMCC)
  - **OCPP** support
  - **Build-your-own**: Phoenix Contact (includes ESL Walli), EVSE DIN
  - **Smart switches**: AVM, Home Assistant, Homematic IP, HomeWizard, myStrom, Shelly, Tasmota, TP-Link. [Read more.](https://docs.evcc.io/en/docs/devices/smartswitches)
  - **Heat pumps and electric heaters:** alpha innotec, Bosch, Buderus, Bösch, CTA, Daikin, Elco, IDM, Junkers, Kermi, Lambda, my-PV, Nibe, Novelan, Roth, Stiebel Eltron, Tecalor, Vaillant, Viessmann, Wolf, Zewotherm [Read more.](https://docs.evcc.io/en/docs/devices/heating)
- support for many [energy meters](https://docs.evcc.io/en/docs/devices/meters):
  - **Solar inverters and battery systems**: A-Tronix, Acrel, Ads-tec, Alpha ESS, Ampere, Anker, APsystems, AVM, Axitec, BGEtech, Bosch, Bosswerk, Carlo Gavazzi, Deye, E3/DC, Eastron, Enphase, FENECON, FoxESS, Fronius, Ginlong, go-e, GoodWe, Growatt, Homematic IP, HomeWizard, Hoymiles, Huawei, IAMMETER, IGEN Tech, Kostal, LG, Loxone, M-TEC, Marstek, myStrom, OpenEMS, Powerfox, Qcells, RCT, SAJ, SAX, SENEC, Senergy, Shelly, Siemens, Sigenergy, SMA, Smartfox, SofarSolar, Solaranzeige, SolarEdge, SolarMax, Solarwatt, Solax, Solinteng, Sonnen, St-ems, Steca, Sungrow, Sunsynk, Sunway, Tasmota, Tesla, TP-Link, VARTA, Victron, Wattsonic, Youless, ZCS Azzurro, Zendure [Read more.](https://docs.evcc.io/en/docs/devices/meters)
  - **General energy meters**: A-Tronix, ABB, Acrel, Alpha ESS, Ampere, AVM, Axitec, Bernecker Engineering, BGEtech, Bosch, Carlo Gavazzi, cFos, Deye, DSMR, DZG, E3/DC, Eastron, Enphase, ESPHome, FENECON, FoxESS, Fronius, Ginlong, go-e, GoodWe, Growatt, Homematic IP, HomeWizard, Huawei, IAMMETER, inepro, IOmeter, Janitza, KEBA, Kostal, LG, Loxone, M-TEC, mhendriks, my-PV, myStrom, OpenEMS, Orno, P1Monitor, Powerfox, Qcells, RCT, Saia-Burgess Controls, SAJ, SAX, Schneider Electric, SENEC, Shelly, Siemens, Sigenergy, SMA, Smartfox, SofarSolar, Solaranzeige, SolarEdge, SolarMax, Solarwatt, Solax, Solinteng, Sonnen, St-ems, Sungrow, Sunsynk, Sunway, Tasmota, Tesla, Tibber, TQ, VARTA, Victron, Volkszähler, Wago, Wattsonic, Weidmüller, Youless, ZCS Azzurro, Zuidwijk. [Read more.](https://docs.evcc.io/en/docs/devices/meters)
  - **Integrated systems**: SMA Sunny Home Manager and Energy Meter, KOSTAL Smart Energy Meter (KSEM, EMxx)
  - **Sunspec**-compatible inverter or home battery devices
  - **mbmd**-compatible devices, see [volkszaehler/mbmd](https://github.com/volkszaehler/mbmd#supported-devices) for a complete list
- [vehicle](https://docs.evcc.io/en/docs/devices/vehicles) integrations (state of charge, remote charge, battery and preconditioning status):
  - Aiways, Audi, BMW, Citroën, Dacia, DS, Fiat, Ford, Hyundai, Jeep, Kia, Mercedes-Benz, MG, Mini, Nissan, Opel, Peugeot, Polestar, Renault, Seat, Skoda, Smart, Tesla, Toyota, Volkswagen, Volvo, Zero Motorcycles. [Read more.](https://docs.evcc.io/en/docs/devices/vehicles)
  - Services: OVMS, Tronity
  - Scooters: NIU
- [plugins](https://docs.evcc.io/en/docs/devices/plugins) for integrating with any charger/ meter/ vehicle:
  - Modbus, HTTP, MQTT, Javascript, WebSockets and shell scripts
- status [notifications](https://docs.evcc.io/en/docs/reference/configuration/messaging) using [Telegram](https://telegram.org), [PushOver](https://pushover.net) and [many more](https://containrrr.dev/shoutrrr/)
- logging using [InfluxDB](https://www.influxdata.com) and [Grafana](https://grafana.com/grafana/)
- [REST](https://docs.evcc.io/en/docs/integrations/rest-api) and [MQTT](https://docs.evcc.io/en/docs/integrations/mqtt-api) APIs for integration with home automation systems
- Add-ons for [Home Assistant](https://docs.evcc.io/en/docs/integrations/home-assistant) and [OpenHAB](https://www.openhab.org/addons/bindings/evcc) (not maintained by the evcc core team)

## Getting Started

You'll find everything you need in our [documentation](https://docs.evcc.io/en/).

## Contributing

Technical details on how to contribute, how to add translations and how to build evcc from source can be found [here](CONTRIBUTING.md).

[![Weblate Hosted](https://hosted.weblate.org/widgets/evcc/-/evcc/287x66-grey.png)](https://hosted.weblate.org/engage/evcc/)

## Sponsorship

<img src="assets/github/evcc-gopher.png" align="right" width="150" />

evcc believes in open source software. We're committed to provide best in class EV charging experience.
Maintaining evcc consumes time and effort. With the vast amount of different devices to support, we depend on community and vendor support to keep evcc alive.

While evcc is open source, we would also like to encourage vendors to provide open source hardware devices, public documentation and support open source projects like ours that provide additional value to otherwise closed hardware. Where this is not the case, evcc requires "sponsor token" to finance ongoing development and support of evcc.

Learn more about our [sponsorship model](https://docs.evcc.io/en/docs/sponsorship).
