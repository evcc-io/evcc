# Templates folder documentation

## Folders

- definition: hold all device templates definitions in yaml files
  - charger: all charger templates
  - meter: all meter templates
  - vehicle: all vehicle templates
- docs: content is generated via `go generate ./...` using the above templates for the evcc documentation page to be used

## Template Documentation

The following describes each possible element in a yaml file

## `template`

`template` expects a unique template name for the current device class (charger, meter, vehicle are device classes)

## `products`

`products` expects a list of products that work with this template.

Each product contains:

- `brand`: an optional brand description of the product
- `description`: an optional description e.g. of the product model. Expects `generic`, `de`, `en`: an optional description of the product

Either `brand`, or `description` need to be set.

## `group`

`group` is used to group switchable sockets and generic device support (e.g. SunSpec) templates.

## `guidedsetup` (Obsolete)

`guidedsetup` is enabled when the device has linked templates or >1 usage. It is used with devices that provide multiple meter usages, or meter devices that are typically installed with specific other devices. Mostly used for meter devices that provided multiple usage data with the same user input. These devices are then sorted at the bottom of the product list.

## `linked`

Allows to define a list of meter devices that are typically installed with this device. Enables `guidedsetup` mode.

#### `template`

`template` expects the linked device `template` value

#### `usage`

`usage` expects the meter usage type this device will be used for

**Possible values**:

- `grid`: for grid meters
- `pv`:  for pv inverter/meter
- `battery`: for battery inverter/meter

#### `multiple`

`multiple:true` to define that multiple devices of this template can be added

#### `excludetemplate`

`excludetemplate` defines a linked device `template` value. If defined and a device of the linked template is added, then this linked template won't be considered in the flow

Example Use Case: With SMA Home Manager, there can be a SMA Energy Meter used for getting the PV generation or multiple SMA PV inverters. But never both together. So if the used added an SMA Energy Meter, then the flow shoudn't ask for SMA PV inverters.

## `capabilities`

`capabilities` provides an option to define special capabilities of the device as a list of strings

**Possible Values**:

- `iso151182`: If the charger supports communicating via ISO15118-2
- `rfid`: If the charger supports RFID
- `1p3p`: If the charger supports 1P/3P-phase switching
- `smahems`: If the device can be used as an SMA HEMS device, only used for the SMA Home Manager 2.0 right now

## `requirements`

`requirements` provides an option to define various requirements / dependencies that need to be setup

### `evcc`

`evcc` is a list of evcc specific system requirements

**Possible Values**:

- `sponsorship`: If the device requires a sponsorship token
- `eebus`: If the device is accessed via the eebus protocol and thus requires the corresponding setup
- `mqtt`: If the device a MQTT setup

### `description`

`description` expects language specific texts via `de`, `en` to provide specific things the user has to do, e.g. minimum firmware versions or specific hardware setup requirements. The content can be multiline and Markdown

## `params`

`params` describes the set of parameters the user needs to provide a value for.

## `preset`

`preset` reference value of a predefined params set defined in `parambaselist.yaml`, so these params don't need to be redefined in each template. The `example` and `default` values for each predefined value can be overwritten.

### `name`

`name` expects a name for the parameter, which will be used in the `render` section to reference the param and provide the user entered value.

**Note**: There a few default `name` values with specific internal meaning and consequences!

**Predefined name values**:

- `usage`: specifies a list of meter classes, the device can be used for. Possible values are `grid`, `pv`, `battery`, and `charger`
- `modbus`: specifies that this device is accessed via modbus. It requires the `choice` property to have a list of possible interface values the device provides. These values can be `rs485` and `tcpip`. The command will use either to ask the appropriate questions and settings. The `render` section needs to include the string `{{include "modbus" .}}` in all places where the configuration needs modbus settings.

#### Usage Options

- `allineone`: Defines if the different usages are all available in a single device. Enables `guidedsetup` mode.

#### Modbus Options

- `id`: Device specific default for modbus ID
- `port`: Device specific default for modbus TCPIP port
- `baudrate`: Device specific default for modbus RS485 baudrate
- `comset`: Device specific default for modbus RS485 comset

### `description`

`description` allows to define user friendly and language specific names via `de`, `en`, `generic`

### `dependencies`

`dependencies` allows to define a list of checks, when this param should be presented to the user, if it should be only in special cases

#### `name`

`name` referenced the `param` `name` value

#### `check`

`check` defines which kind of check should be performed

**Possible values**:

- `empty`: if the `value` of the referenced `param` `name` should be empty
- `notempty`: if the `value` of the referenced `param` `name` should be **NOT** empty
- `equal`: if the `value` of the referenced `param` `name` should match the value of the `value` property

#### `value`

`value` property is used in the `equal` `check`

### `required`

`required: true` defines if the user has to provide a value. Default is `false`

### `mask`

`mask: true` defines if the user input should be masked, e.g. for passwords. Defaut is `false`

### `default`

`default` defines a default value to be used. For these cases the user can then simply press enter in the CLI.

### `example`

`example` provides an example value, so the user can get an idea of what is expected and what to look out for

### `type`

`type` allows to define the value type to let the CLI verify the user provided content

**Possible values**:

- `string`: for string values (default)
- `bool`: for `true` and `false` values. If `help` is provided, than that help text is presented as the question
- `int`: for int values
- `float`: for float values
- `list`: for a list of strings, e.g.used for defining a list of `identifiers` for `vehicles`
- `chargemodes`: for a selection of charge modes (including `None` which results in the param not being set)

### `advanced`

`advanced` allows to specify if the param should only be asked if the cli is run with `--advanced`. Mostly used for non required params that are meant for users with advanced needs and knowledge.

### `help`

`help` expects language specific help texts via `generic` (language independent), `de`, `en`

## `render`

`render` contains the internal device configuration. All `param` `name` values can be used as a template variable, e.g. `{{ .host }}` for a param named `host`. The content is a go template, so all of go template feature can be used, e.g. `{{- if ... }}` statements, etc.
