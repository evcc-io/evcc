# Simulator
The simulator allows you to test the evcc control mechanisms by replacing real devices - such as a meter or a vehicle by simulated "sim-devices".  
The simulator itself coordinates the sim devices and executes a program that can be defined in the simulator section of the evcc.yaml file.

## sim-devices
The following **sim-devices** are provided:
* **sim-vehicle**: simulates a vehicle and its battery. Bidirectional charging can be activated.
* **sim-charger**: simulates a charger to which vehicles can be connected
* **sim-switch1p3p**: simulates a phase switch that can switch a charger from 1phase to 3phases. The name of the switch must be configured in the switch1p3p config entry of the charger to which it belongs
* **sim-meter**: simulates different types of meters. The following **usage** types are supported:
  * **grid**: simulates a grid-meter (is always required - calculation is done by the simulator itself). Only one meter with usage "grid" is allowed.
  * **pv**: simulates a pv. A free number of "pv" meters is allowed (one for each pv)
  * **battery**: simulates a battery. A free number of "battery" meters are allowed (one for each battery)
  * **home**: simulates the power consumption of your home - the value can be controlled using your simulation program. A free number of "home" meters is allowed - but the sense can be questioned

A mixture of real devices and sim-devices is not supported.

## Configuration evcc.yaml
The various meters, charger, vehicles, switches and the simulator must be configured in the .yaml file for the simulator to work correctly.

**ATTENTION**: don't forget to add the simulator to the site configuration - for details see below

### site
For the simulator to be updated in the site update cycle its name must be set in the site configuration.
The name of the simulator is defined in the simulator node of the simulators configuration.
```yaml
site:
  ...
  simulators:      # list of simulators that shall run for this site
    - simSimulator # name of the simulator
```
### sim-meter "grid"
Simulates the grid node. Exactly one sim-meter with usage "grid" must exist for the simulator to work correctly
```yaml
meters:
  - name: gridSim   # name can be freely chosen
    type: sim-meter # simulation meter
    usage: grid
```

### sim-meter "pv"
Simulates a pv node. A free number of pv nodes can be added. 
```yaml
meters: 
  - name: pvSim     # name can be freely chosen
    type: sim-meter # simulation meter
    usage: pv
    power: 10000    # [w] initial power the pv generates (positive sign)
  - name: pvSim2    # name can be freely chosen (must be unique)
    type: sim-meter # simulation meter
    usage: pv
    power: 1500    # [w] initial power the pv generates (positive sign)
```

### sim-meter "home"
Simulates the power consumption of the home. A free number of home nodes can be added - but this typically doesn't make sense.
```yaml
meters: 
  - name: homeSim   # name can be freely chosen
    type: sim-meter # simulation meter
    usage: home
    power: -400     # [w] initial power consumption (negative sign) of the home node
```
### sim-meter "battery"
Simulates a battery node. A free number of battery nodes can be added.
```yaml
meters:
  - name: batterySim # name can be freely chosen
    type: sim-meter
    usage: battery
    capacity: 8      # [kWh] capacity of the battery - as integer (no float value allowed)
    powerLimit: 5000 # [W] maximum charging and discharging power this battery supports
    soc: 65          # [%] initial state of charge of the battery

```
### sim-switch1p3p
Simulates a phases switch that allows evcc to switch between 1phase and 3phases for charging.
```yaml
switches1p3p:
  - name: switch1p3pSim
    type: sim-switch1p3p
    phases: 1 # initial phase setting
```

### sim-charger
Simulates one charger. A free number of chargers can be added.
```yaml
chargers:
  - name: chargerSim  # name can be freely chosen
    type: sim-charger
    status: A         # initial charger status. See api.ChargerStatus
    switch1p3p: switch1p3psim # name of the 1p3p switch that controls the phases of this charger. If no switch is assigned the charger does not support 1p3p switching
```
### sim-vehicle
Simulates a vehicle and its battery. A free number of vehicles can be added.
```yaml
vehicles:
  - name: vehicleSim     # name can be freely chosen
    type: sim-vehicle
    capacity: 58         # [kWh] - integer, capacity of the vehicle battery
    powerLimit: 11000    # [w] maximum charge/discharge power of the vehicle
    bidirectional: false # Set to true to simulate "Vehicle2Grid"
    soc: 48              # [%] initial state of charge of the vehicle battery
    title: mySimCar      # title of the vehicle
    identifiers:         # list of identifiers for this vehicle
      - Identifier1
      - Identifier2
```
### simulator
The simulator ist he core component of the simulation. It interacts with the sim-devices, calculates the grid meter values and executes the simulator program that allows for example to connect/disconnect a vehicle, change the power consumption of the "home" node, or change the power generation of a "pv" node. It also supports changing the active charge-mode of the Gui. At maximum one simulator command is executed each simulator "update" (which is also the site update cycle time).
A free number of simulators can be configured - but currently only one simulator that "does it all" is in use - it is therefore not needed to configure multiple simulators.
```yaml
simulators:
 - name: simSimulator   # name can be freely chosen
    type: simulator
    active: true        # set to false to deactivate the simulator
    program:            # simple example program
      - cmd: connect        # command to connect a vehicle to a charger
        object: chargerSim  # the charger to which the vehicle shall be connected
        attribute: vehicle
        value: vehicleSim   # the name of the vehicle that shall be connected
      - cmd: sleep          # commands the simulation to sleep for the given value
        value: 4s           # time for sleeping before the next command is executed
      ...
```
### Simulator Program syntax
The simulator program consists of a list of commands. Each command can have the following elements. The avialable elements and their interpretation depends on the command. Currently only a very limited set of commands is supported. They can be easily extended within the simulator.go file.
```yaml
  - cmd: <commandName>         # the command to execute
    object: <objectName>       # the name of the object on which to execute the command
    attribute: <attributeName> # the name of the attribute of the object that shall be changed or tested
    value: <value>             # the value of the attribute to change or test
    timeout: <value>           # a timeout value for the command
```
#### sleep
Tells the simulator to "sleep" for the given time.
After the sleep time elapsed the next simulator command is executed.
```yaml
  - cmd: sleep  
    value: 4s   # time to sleep before the next simulator command is executed  
```
#### connect
Tells the simulator to connect a given vehicle to a given charger.
```yaml
  - cmd: connect
    object: chargerSim # the name of the charger to which the vehicle shall be connected
    attribute: vehicle # change the vehicle attribute of the charger
    value: vehicleSim  # name of the vehicle to connect to the charger
```
#### disconnect
Tells the simulator to disconnect a vehicle from a given charger.
The simulator does not complain if no vehicle is connected to the charger.
```yaml
  - cmd: disconnect
    object: chargerSim # the name of the charger from which the vehicle shall be disconnected
```
#### set
Tells the simulator to set an attribute of a given sim-device.
```yaml
  - cmd: set
    object: homeSim  # the name of the sim-device whose value shall be changed
    attribute: power # currently only "power" is supported ... but it is quite simple to extend the simulator to support "soc"
    value:  -1250    # the value to set for the homeSim power attribute
```
#### setGui
Tells the simulator to execute a GUI user interaction.
Currently only the "mode" is supported.
```yaml
  - cmd: setGui
    object:         # not yet supported - should be used to select the loadpoint. Currently the very first loadpoint is used
    attribute: mode # to change the charge mode of the loadpoint
    value: off      # allowed values: "off": to stop charging, "now": to start charging with max current, "minpv" to start charging with min current + pv excess, "pv" to start charging when pv production is sufficient
```
#### expect
Tells the simulator to check the value of an attribute within a given time window. The simulation program is stopped when the attribute doesn't reach the value. The simulation program switches to the next command as soon as the value is reached - independent of the time window. E.g. when the time window is set to 60s and the value is reached after 10s the simulator goes to the next command after 10s.
```yaml
  - cmd: expect
    object: gridSim  # name of the sim device whose attribute value shall be checked
    attribute: power # currently only this attribute is supported
    value: -1200     # the expected value
    timeout: 20s     # maximum time the simulator checks the expected value.
                     # if the  the simulator 
```
## Behind the scenes
During initialization the simulator is initialized after all devices to be able to search the list of devices and find all sim-devices with which it can operate.
The sim-devices are identified by the implementation of the "Sim" interface.
The simulator ist updated by the update loop of the site.
The simulator uses the REST API to change the GUI settings such as the charge-mode.