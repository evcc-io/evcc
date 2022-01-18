# evcc 🚘☀️

[![Open in Visual Studio Code](https://open.vscode.dev/badges/open-in-vscode.svg)](https://open.vscode.dev/evcc-io/evcc)
[![Build Status](https://github.com/evcc-io/evcc/workflows/Build/badge.svg)](https://github.com/evcc-io/evcc/actions?query=workflow%3ABuild)
[![Code Quality](https://goreportcard.com/badge/github.com/evcc-io/evcc)](https://goreportcard.com/report/github.com/evcc-io/evcc)
[![Latest Version](https://img.shields.io/github/release/evcc-io/evcc.svg)](https://github.com/evcc-io/evcc/releases)
[![Pulls from Docker Hub](https://img.shields.io/docker/pulls/andig/evcc.svg)](https://hub.docker.com/r/andig/evcc)
[![Donate](https://img.shields.io/badge/Donate-PayPal-green.svg)](https://www.paypal.com/cgi-bin/webscr?cmd=_s-xclick&hosted_button_id=48YVXXA7BDNC2)

Evcc is an extensible EV Charge Controller with PV integration implemented in [Go][2]. Featured in [PV magazine](https://www.pv-magazine.de/2021/01/15/selbst-ist-der-groeoenlandhof-wallbox-ladesteuerung-selbst-gebaut/).

![Screenshot](docs/screenshot.png)

## Features

- simple and clean user interface
- multiple [chargers](https://docs.evcc.io/docs/devices/chargers):
  - Open source: [openWB](https://openwb.de/), [EVSEWifi](https://www.evse-wifi.de) (includes smartWB)
  - Other commercial: ABL eMH1, Alfen Eve, cFos PowerBrain, Daheimladen, go-eCharger, Heidelberg Energy Control, Innogy eBox, KEBA/BMW, NRGkick, PC Electric (includes Garo), Wallbe, Mobile Charger Connect, EEBUS (experimental)
  - Build-your-own: Phoenix (includes ESL Walli), [SimpleEVSE](https://www.evse-wifi.de/produkt-schlagwort/simple-evse-wb/)
  - Smart-Home outlets: FritzDECT, Shelly, Tasmota, TP-Link
- multiple [meters](https://docs.evcc.io/docs/devices/meters): ModBus (Eastron SDM, MPM3PM, SBC ALE3 and many more), Discovergy (using HTTP plugin), SMA Sunny Home Manager and Energy Meter, KOSTAL Smart Energy Meter (KSEM, EMxx), any Sunspec-compatible inverter or home battery devices (Fronius, SMA, SolarEdge, KOSTAL, STECA, E3DC, ...), Tesla PowerWall, LG ESS HOME
- wide support of vendor-specific [vehicles](https://docs.evcc.io/docs/devices/vehicles) interfaces (remote charge, battery and preconditioning status): Audi, BMW, Fiat, Ford, Hyundai, Kia, Mini, Nissan, Niu, Porsche, Renault, Seat, Smart, Skoda, Tesla, Volkswagen, Volvo, Tronity
- [plugins](https://docs.evcc.io/docs/reference/plugins) for integrating with any charger/ meter/ vehicle: Modbus (meters and grid inverters), HTTP, MQTT, Javascript, WebSockets and shell scripts
- status [notifications](https://docs.evcc.io/docs/reference/configuration/messaging) using [Telegram](https://telegram.org), [PushOver](https://pushover.net) and [many more](https://containrrr.dev/shoutrrr/)
- logging using [InfluxDB](https://www.influxdata.com) and [Grafana](https://grafana.com/grafana/)
- granular charge power control down to mA steps with supported chargers (labeled by e.g. smartWB als [OLC](https://board.evse-wifi.de/viewtopic.php?f=16&t=187))
- REST and MQTT [APIs](https://docs.evcc.io/docs/reference/api) for integration with home automation systems (e.g. [HomeAssistant](https://github.com/evcc-io/evcc-hassio-addon))

## Getting Started

You'll find everything you need in our [documentation](https://docs.evcc.io/) (German).

## Build

To build EVCC from source, [Go][2] 1.16 and [Node][3] 16 are required:

```sh
make
```

## Sponsorship

Evcc believes in open source software. We're committed to provide best in class EV charging experience.
Maintaining evcc consumes time and effort. With the vast amount of different devices to support, we depend on community and vendor support to keep EVCC alive.

While evcc is open source, we would also like to encourage vendors to provide open source hardware devices, public documentation and support open source projects like hours that provide additional value to otherwised closed hardware. Where this is not the case, EVCC requires "sponsor token" to finance ongoing development and support of evcc.

The personal sponsor token requires a [Github Sponsorship](https://github.com/sponsors/andig) and can be requested at [cloud.evcc.io](https://cloud.evcc.io/). A sponsor token is valid for one year and can be renewed any time with active sponsorship.

## Background

<img src="docs/logo.png" align="right" width="150" />

Evcc is heavily inspired by [OpenWB][1]. However, in 2019, I found OpenWB's architecture slightly intimidating with everything basically global state and heavily relying on shell scripting. On the other side, especially the scripting aspect is one that contributes to OpenWB's flexibility.

Hence, for a simplified and stricter implementation of an EV charge controller, the design goals for evcc were:

- typed language with ability for systematic testing - achieved by using [Go][2]
- structured configuration - supports YAML-based [config file](evcc.dist.yaml)
- avoidance of feature bloat, simple and clean UI - utilizes [Vue.js][4] and [Bootstrap][5]
- containerized operation beyond Raspberry Pi - provide multi-arch [Docker Image][6]

[1]: https://github.com/snaptec/openWB
[2]: https://golang.org
[3]: https://nodejs.org/
[4]: https://vuejs.org
[5]: https://getbootstrap.org
[6]: https://hub.docker.com/repository/docker/andig/evcc
