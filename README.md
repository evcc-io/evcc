# evcc

[![Build Status](https://travis-ci.org/andig/evcc.svg?branch=master)](https://travis-ci.org/andig/evcc)

EVCC is an extensible EV Charge Controller with PV integration implemented in [Go](2).

### Features

- simple and clean user interface
- multiple [chargers](#charger): Wallbe (tested with Wallbe Eco S), Phoenix controllers (similar to Wallbe), any other charger using scripting
- more chargers experimentally supported: NRGKick, Go-E, SimpleEVSE, EVSEWifi
- different [vehicles](#vehicle) to show battery status: Audi (eTron), BMW (i3), Tesla, Nissan (Leaf), any other vehicle using scripting
- integration with home automation - supports shell scripts and MQTT
- status notifications using [Telegram](https://telegram.org) and [PushOver](https://pushover.net)
- logging using [InfluxDB](https://www.influxdata.com)
- soft ramp-up/ramp-down of charge current ensures contactor only switched at minimum current
- electric contactor protection
- REST API

![Screenshot](docs/screenshot.png)

### Background

EVCC is heavily inspired by [OpenWB](1). However, I found OpenWB's architecture slightly intimidating with everything basically global state and heavily relying on shell scripting. On the other side, especially the scripting aspect is one that contributes to [OpenWB's](1) flexibility.

Hence, for a simplified and stricter implementation of an EV charge controller, the design goals for EVCC were:

- typed language with ability for systematic testing - achieved by using [Go](2)
- structured configuration - supports YAML-based [config file](evcc.dist.yaml)
- avoidance of feature bloat, simple and clean UI - utilizes [Bootstrap](3)
- containerized operation beyond Raspberry Pi - provide multi-arch [Docker Image](4)
- support for multiple load points - tbd

### Note

EVCC comes without any guarantee. You are using this software **entirely** at your own risk. It is your responsibility to verify it is working as intended.
EVCC requires a supported charger and a combination of grid, PV and charge meter.
Charger and meters **must** be installed by a certified professional.

## Installation

EVCC is provided as binary executable file and docker image. Download the file for your platform and then execute like this:

    evcc -h

or to run EVCC with given config file and UI on port 7070 using Docker:

    docker run -v $(pwd)/evcc.dist.yaml:/etc/evcc.yaml -p 7070:7070 andig/evcc -h

To build EVCC from source, [Go](2) 1.13 is required:

    make

## Configuration

### Charge Modes

Multiple charge modes are supported:

- **Off**: disable the charger, even if car gets connected.
- **Now** (**Sofortladen**): charge immediately with maximum allowed current.
- **Min + PV**: charge immediately with minimum configured current. Additionally use PV if available.
- **PV**: use PV as available. May not charge the car if PV remains dark.

In general, due to the minimum value of 5% for signalling the EV duty cycle, the charger cannot limit the current to below 6A. If the available power calculation demands a limit less than 6A, handling depends on the charge mode. In **PV** mode, the charger will be disabled until available PV power supports charging with at least 6A. In **Min + PV** mode, charging will continue at minimum current of 6A and charge current will be raised as PV power becomes available again.

### PV generator configuration

For both PV modes, EVCC needs to assess how much residual PV power is available at the grid connection point and how much power the charger actually uses. Various methods are implemented to obtain this information, with different degrees of accuracy.

- **PV meter**: Configuring a *PV meter* is the simplest option. *PV meter* measures the PV generation. The charger is allowed to consume:

      Charge Power = PV Meter Power - Residual Power

  The *Residual Power* is a configurable assumption how much power remaining facilities beside the charger use.

- **Grid meter**: Configuring a *grid meter* is the preferred option. The *grid meter* is expected to be a two-way meter (import+export) and return the current amount of grid export as negative value measured in Watt (W). The charger is then allowed to consume:

      Charge Power = Current Charge Power - Grid Meter Power - Residual Power

  In this setup, *residual power* is used as margin to account for fluctuations in PV production that may be faster than EVCC's control loop.

### Charger configuration

When using a *grid meter* for accurate control of PV utilization, EVCC needs to be able to determine the current charge power. There are two configurations for determining the *current charge power*:

- **Charge meter**: A *charge meter* is often integrated into the charger but can also be installed separately. EVCC expects the *charge meter* to supply *charge power* in Watt (W) and preferably *total energy* in kWh.
If *total energy* is supplied, it can be used to calculate the *charged energy* for the current charging cycle.

- **No charge meter**: If no charge meter is installed, *charge power* is deducted from *charge current* as controlled by the charger. This method is less accurate than using a *charge meter* since the EV may chose to use less power than EVCC has allowed for consumption.
If the charger supplies *total energy* for the charging cycle this value is preferred over the *charge meter*'s value (if present).

## Implementation

EVCC consists of four basic elements: *Charger*, *Meter*, *SoC* and *Loadpoint*. Their APIs are described in [api/api.go](https://github.com/andig/evcc/blob/master/api/api.go)

### Charger

Charger is responsible for handling EV state and adjusting charge current:

- `Status()`: get charge controller status (`A...F`)
- `Enabled()`: get charger availability
- `Enable()`: set charger availability
- `MaxCurrent()`: set maximum allowed charge current in A

Optionally, charger can also provide:

- `CurrentPower()`: power in W (used if charge meter is not present)

Available charger implementations are:

- `wallbe`: Wallbe Eco chargers (see [Hardware Preparation](#Wallbe-Hardware-Preparation) for preparing the Wallbe)
- `phoenix`: chargers with Phoenix controllers
- `simpleevse`: chargers with SimpleEVSE controllers connected via ModBus (e.g. OpenWB)
- `evsewifi`: chargers with SimpleEVSE controllers using [SimpleEVSE-Wifi](https://github.com/CurtRod/SimpleEVSE-WiFi)
- `nrgkick`: NRGKick chargers with Connect module
- `go-e`: go-eCharger chargers
- `default`: default charger implementation using configurable [plugins](#plugins) for integrating any type of charger

#### Wallbe Hardware Preparation

Wallbe chargers are supported out of the box. The Wallbe must be connected using Ethernet. If not configured, the default address `192.168.0.8:502` is used.

To allow controlling charge start/stop, the Wallbe physical configuration must be modified. This requires opening the Wallbe. Once opened, DIP 10 must be set to ON:

![dip10](docs/dip10.jpeg)

More information on interacting with Wallbe chargers can be found at [GoingElectric](https://www.goingelectric.de/forum/viewtopic.php?p=1212583). Use with care.

**NOTE:** Opening the wall box **must** only be done by certified professionals. The box **must** be disconnected from mains before opening.

### Meter

Meters provide data about power and energy consumption:

- `CurrentPower()`: power in W
- `TotalEnergy()`: energy in kWh (optional)

Meter has a single implementation where meter readings- power and energy- can be configured to be delivered by [plugin](#plugins).

### Vehicle

Vehicle represents a specific EV vehicle and its battery:

- `Title()`: vehicle name for display in the configuration UI
- `Capacity()`: battery capacity in kWh
- `ChargeState()`: state of charge in %

Optionally, vehicles can also provide:

- `CurrentPower()`: charge power in W (used if charge meter not present)
- `ChargedEnergy()`: charged energy in kWh
- `ChargeDuration()`: charge duration

If vehicle is configured and assigned to the charger, charge status and remaining charge duration become available in the user interface.

Available vehicle implementations are:

- `audi`: Audi (eTron)
- `bmw`: BMW (i3)
- `nissan`: Nissan (Leaf)
- `tesla`: Tesla (any model)
- `default`: default vehicle implementation using configurable [plugins](#plugins) for integrating any type of vehicle

### Plugins

Plugins are used to implement accessing and updating generic data sources. EVCC supports the following *read/write* plugins:

- `mqtt`: this plugin allows to read values from MQTT topics. This is particularly useful for meters, e.g. when meter data is already available on MQTT. See [MBMD](5) for an example how to get Modbus meter data into MQTT.
This plugin type is read-only and does not provide write access.

  Sample configuration:

      type: mqtt
      topic: mbmd/sdm1-1/Power
      timeout: 30s

- `script`: the script plugin executes external scripts to read or update data. This plugin is useful to implement any type of external functionality.

  Sample configuration:

      type: script
      cmd: /bin/bash -c "echo 50"
      timeout: 5s

When using plugins for *write* access, the actual data is provided as variable in form of `${var[:format]}`. The variable is replaced with the actual data before the plugin is executed.

[1]: https://github.com/snaptec/openWB
[2]: https://golang.org
[3]: https://getbootstrap.org
[4]: https://hub.docker.com/repository/docker/andig/evcc
[5]: https://github.com/volkszaehler/mbmd
