# Load management

## Use cases

- I want to avoid avoid overload on main circuit due to one or more load points in use
- I want to run multiple load points on a restricted circuit (outside load place with limited power supply)
- I generally want to limit the consumption of all or some load points

## Goals

- hierarchical circuits to model the real infrastructure
- secure infrastructure (wires, fuses) using current limit values
- restrict general power consumption using power value
- allow dedicated meters per circuit if present in installation
- support circuits without meters (usually sub circuits in private installations don't have dedicated meters) or virtually combine multiple loadpoints to a circuit irrespecitve of their physical distribution
- co-exists with existing charge modes (pv, min, now, off)

## Non Goals / Out of Scope

- Load balancing. We use "first come first serve".

## Remarks

- Load management for current checking is based on current per phase, while pv modes are based on power
- We don't have phase accurate modeling (installation tracked down for each phase completely. Multiple load points might be connected using phase rotation to avoid shear load)
- As reference we use always the highest loaded phase per circuit. This might lead to not optimal usage of available current and power.
- Power limiting is based on power values from meters and LPs
- No load balancing means that we have "first come first serve" situation.
- If a circuit has no meter, the consumption will be evaluated from connected consumers (virtual meter) and sub circuits. If a load point has a real meter configured, the current from phase 1 is used.
- With virtual meter: If the load point has no meter or does not provide the phase currents, the assigned `chargeCurrent` resp `chargePower` is used. Chargers or vehicles which do not set the state accordingly after charging, might reserve up to `maxCurrent()` in the circuit, which could prevent other circuits to start charging.

## Implementation requirements

- separate module for simple testing
- isolation using interfaces
- virtual meter for consistent circuit logic

# Usage

## Configuration

Configuration assistant supports circuit creation in advanced mode, up to 4 levels in hierarchy. Manually there is no limit in circuit depth.

## Example 1: all load points shall not use more than 25A

- no load management for main circuit, only for garage
- max current is determined by load point consumption

```
circuits:
- name: Garage
  maxCurrent: 25
  meter:
  parent:

loadpoints:
- title: Garage links
  charger: wallbox5
  mode: off
  phases: 3
  mincurrent: 6
  maxcurrent: 16
  resetOnDisconnect: false
  circuit: Garage
- title: Garage rechts
  charger: wallbox6
  mode: off
  phases: 3
  mincurrent: 6
  maxcurrent: 16
  resetOnDisconnect: false
  circuit: Garage
```

## Example 2: secure main circuit, all load points shall not exceed 16A

- main circuit is covered.
- max current is determined by load point consumption

```
circuits:
- name: main
  maxCurrent: 35
  meter: grid1
  parent:
- name: Garage
  maxCurrent: 16
  meter:
  parent: main

loadpoints:
- title: Garage links
  charger: wallbox5
  mode: off
  phases: 3
  mincurrent: 6
  maxcurrent: 16
  resetOnDisconnect: false
  circuit: Garage
- title: Garage rechts
  charger: wallbox6
  mode: off
  phases: 3
  mincurrent: 6
  maxcurrent: 16
  resetOnDisconnect: false
  circuit: Garage

site:
  title: Mein Zuhause
  meters:
    grid: grid1
    pvs:
    - pv2
    batteries:
    - battery3
```

## Example 3: secure main circuit, use series of load points with separate meter

- main circuit is covered
- consumption of loadpoints is determined using a dedicated meter in this circuit

```
meters:
- type: template
  template: eastron
  id: 1
  host: 1.2.3.4
  port: 502
  usage: grid
  modbus: rs485tcpip
  name: meter_aussen

circuits:
- name: main
  maxCurrent: 63
  meter: grid1
  parent:
- name: Parkplatz
  maxCurrent: 35
  meter: meter_aussen
  parent: main

loadpoints:
- title: Parken 1
  circuit: Parkplatz
  ...
- title: Parken 2
  circuit: Parkplatz
  ...
- title: Parken 3
  circuit: Parkplatz
  ...
- title: Parken 4
  circuit: Parkplatz
  ...

site:
  title: Arztpraxis Feelgood
  meters:
    grid: grid1
```

## Example 4: limit main grid use by max power

- make sure not drawing more than 4kW from grid
- still check fuses on 20A
- max current is determined by load point consumption

```
circuits:
- name: Main
  maxCurrent: 20
  maxPower: 4
  meter: grid
  parent:
- name: Garage
  maxCurrent: 25
  meter:
  parent: main

loadpoints:
- title: Garage links
  charger: wallbox5
  mode: off
  phases: 3
  mincurrent: 6
  maxcurrent: 16
  resetOnDisconnect: false
  circuit: Garage
- title: Garage rechts
  charger: wallbox6
  mode: off
  phases: 3
  mincurrent: 6
  maxcurrent: 16
  resetOnDisconnect: false
  circuit: Garage
```

## Example 5: restrict all LP to 11kW total, irrespective of phase currents

```
circuits:
- name: Chargers
  maxCurrent: 0 # disable current Checking
  maxPower: 11
  meter:
  parent:
loadpoints:
- title: Garage links
  charger: wallbox5
  mode: off
  phases: 3
  mincurrent: 6
  maxcurrent: 16
  resetOnDisconnect: false
  circuit: Chargers
- title: Garage rechts
  charger: wallbox6
  mode: off
  phases: 3
  mincurrent: 6
  maxcurrent: 16
  resetOnDisconnect: false
  circuit: Chargers
```

## Implementation

### Circuit

Using circuit struct with

- max current: highest allowed current, will not be checked when 0 in config
- max power: maximum power in circuit, will not be checked when 0 in config
- sub circuits: if sub circuits are connected
- parent circuit: required to evaluate the remaining current in a hierarchy
- meter: get consumption in circuit
  - if only using power limiting, any meter is ok
  - if using current limiting, a phase current enabled meter needs to be used

Circuit needs to provide on request the `GetRemainingCurrent()` and `CurrentPower()`, defined as `maxCurrent - consumption`. Since a circuit might be included in a hierarchy, the upper circuits might have less remaining current than the actual circuit. Therefore the parent circuit reference is required to get the remaining current of the parent.

Consumption is taken from the assigned meter. A meter is either a physical meter provided by config or a virtual meter (see below).

### Virtual Meter

For circuits without real meter the circuit creates a virtual meter. A virtual meter evaluates the consumption using a list of consumers (load points). If a virtual meter is used, it also uses the sub cicruits as consumer to consider their consumtion.
A virtual meter does not know the load of other consumers of this circuit which are eventually connected. This has to be considered in the limit setting.

### Consumer

A consumer as new interface to let a virtual meter get the current consumption. The load point implements this interface. Load point uses `EffectiveCurrent()` to determine the current consumption.
It also helps the load point to adjust the remaining current when setting new limit.

### Load point

The load points hold a optional reference to one circuit. The cicuit is used to get the remaining current of this circuit when setting the new limit.

## Operation

The circuits are generally passive. On `SetLimit()` of a load point the load point checks the circuit for the remaining current at the beginning and adjusts this if its lower than the requested current.
Since the circuit has the total consumption as base for the remaining current, the returned value includes the current consumption of this load point already. Load point adjusts the remaining current using the consumer interface `MaxPhasesCurrent()`.

In case the remaining current is lower than `MinCurrent`, `SetLimit()` handles this already in the subsequent code of `setLimit()`

## Open Points

- slow reacting chargers might cause interferences on charging (on/off changes)

## Tasks

[x] config assistant for circuits

[x] power limiting

[X] introduce virtual meters to handle circuits w/o real meters

[x] influx values with tag `circuit`
