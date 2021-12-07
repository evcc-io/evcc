# Templates folder documentation

## Folders

- defintion: hold all device templates definitons in yaml files
  - charger: all charger templates
  - meter: all meter templates
  - vehicle: all vehicle templates
- docs: content is generated via `go generate ./..` using the above templates for the evcc documentation page to be used

## Template Documentation

The following describes each possible element in a yaml file

## `template`

`template` expects a unique template name for the current device class (charger, meter, vehicle are device classes)

## `description`

`description` expects a human readable description of the device, best language neutral and containing the name of the product

## `generic`

`generic: true` defines templates that are typically not a hardware product, but rather generic implementations that can be used by a variety of products (e.g. inverters with sunspec support) or for software components like vzlogger.

## `guidedsetup`

`guidedsetup` if the device can be used for a guided setup, which are devices that provide multiple meter usages, or meter devices that are typically installed with specific other devices. Mostly used for meter devices that provided multiple usage data with the same user input. These devices are then sorted at the bottom of the product list.

### `enable`

`enable: true` to define that this device can be used for `guidedsetup`

### `linked`

Allows to define a list of meter devices that are typically installed with this device

#### `template`

`template` expects the linked device `template` value

#### `usage`

`usage` expects the meter usage type this device will be used for

**Possible values**:

- `grid`: for grid meters
- `pv`:  for pv inverter/meter
- `battery`: for battery inverter/meter

## `requirements`

`requirements` provides an option to define various requirements / dependencies that need to be setup

**Possible Values**:

- `sponsorshipt: true`: If the device requires a sponsorship token
- `eebus: true`: If the device is accessed via the eebus protocol and thus requires the corresponding setup
- `hems: sma`: If the device can be used as an SMA HEMS device, only used for the SMA Home Manager 2.0 right now
- `description`: expects language specific texts via `de`, `en` to provide specific things the user has to do, e.g. minimum firmware versions or specific hardware setup requirements
- `uri`: a link providing more help on the requirements

## `loglevel`

`loglevel` defindes the name that can be used in the `levels` configuration for adjusting the log level of individual devices/components/...

## `paramsbase`

`paramsbase` allows to use a predefined set of params, so they don't need to be redefined in each template. The `example` and `default` values for each predefined value can be overwritten.

**Possible values**:

- `vehicle`: Provides a set of params that are used in most vehicles

## `params`

`params` describes the set of parameters the user needs to provide a value for.

### `name`

`name` expects a name for the parameter, which will be used in the `render` section to reference the param and provide the user entered value.

**Note**: There a few default `name` values with specific internal meaning and consequences!

**Predefined name values**:

- `usage`: specifies a list of meter classes, the device can be used for. Possible values are `grid`, `pv`, `battery`, and `charger`
- `modbus`: specifies that this device is accessed via modbus. It requires the `choice` property to have a list of possible interface values the device provides. These values can be `rs485` and `tcpip`. The command will use either to ask the appropriate questions and settings. The `render` section needs to include the string `{{include "modbus" .}}` in all places where the configuration needs modbus settings.

### `required`

`required: true` defines if the user has to provide a value. Default is `false`

### `mask`

`mask: true` defines if the user input should be masked, e.g. for passwords. Defaut is `false`

### `default`

`default` defines a default value to be used. For these cases the user can then simply press enter in the CLI.

### `example`

`example` provides an example value, so the user can get an idea of what is expected and what to look out for

### `valuetype`

`valuetype` allows to define the value type to let the CLI verify the user provided content

**Possible values**:

- `string`: for string values (default)
- `bool`: for `true` and `false` values. If `help` is provided, than that help text is presented as the question
- `number`: for int values
- `float`: for float values
- `stringlist`: for a list of strings, e.g.used for defining a list of `identifiers` for `vehicles`

### `advanced`

`advanced` allows to specify if the param should only be asked if the cli is run with `--advanced`. Mostly used for non required params that are meant for users with advanced needs and knowledge.

### `help`

`help` expects language specific help texts via `de`, `en`

## `render`

`render` contains the internal device configuration. All `param` `name` values can be used as a template variable, e.g. `{{ .host }}` for a param named `host`. The content is a go template, so all of go template feature can be used, e.g. `{{- if ... }}` statements, etc.
