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
- `eebus`: If the device is accessed via the EEBus protocol and thus requires the corresponding setup
- `mqtt`: If the device a MQTT setup

### `description`

`description` expects language-specific texts via `de`, `en` to provide specific things the user has to do, e.g. minimum firmware versions or specific hardware setup requirements.

**Markdown formatting**:

- The content can be multiline
- The content supports Markdown formatting
- External URLs should always use Markdown link format with the hostname as display text: `[docs.example.com](https://docs.example.com/path/to/page)`. This provides clear context while keeping the text readable.
- Use code formatting `` `text` `` for technical identifiers, tokens, configuration values, and entity patterns
- Use bold formatting `**text**` sparingly and only for important warnings or critical information

Example:

```
en: |
  Requires `hcaptcha` token from [developer.example.com](https://developer.example.com/tokens).

  **Attention**: Token is only valid for 2 minutes.
```

## `auth`

`auth` defines OAuth authentication configuration for devices that require user authorization. When specified, the UI OAuth flow and token management are handled automatically. The auth endpoint is called when all required parameters are filled and is re-called on every parameter change.

### `type`

`type` specifies the OAuth provider type. This must reference a dedicated OAuth implementation.

**Available types**: `homeassistant`, `ford-connect`, `viessmann`, `cardata`, `volvo-connected`

### `params`

`params` is a list of parameter names (from the `params` section) that are required for the OAuth configuration. These parameters will be passed to the authentication provider when initiating the OAuth flow. Once all listed parameters have values, the authorization is prepared and the UI displays a redirect link to the external service and device code (if applicable). The preparation is re-triggered whenever any parameter value changes.

**Example**:

```yaml
auth:
  type: viessmann
  params: [clientid, redirecturi, gateway_serial]
```

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

`mask: true` defines if the user input should be masked in the UI (password field). Used for sensitive credentials like passwords, tokens, and API keys that should be hidden from view. Default is `false`.

**Note**: Cannot be used together with `private`.

### `private`

`private: true` marks a parameter as containing personal data (e.g., email addresses, VIN numbers, MAC addresses, locations). This data will be redacted from bug reports and diagnostic information but is visible in the UI. Default is `false`.

**Examples of private data**: usernames, email addresses, VIN, URI, MAC addresses, latitude/longitude, serial numbers

**Note**: Cannot be used together with `mask`.

### `default`

`default` defines a default value to be used, which will be pre-filled in the configuration UI.

### `example`

`example` provides an example value, so the user can get an idea of what is expected and what to look out for

### `type`

`type` allows to define the value type to let the UI verify the user provided content

**Possible values**:

- `string`: for string values (default)
- `bool`: for `true` and `false` values
- `choice`: for a selection from predefined options (defined in `choice` property)
- `chargemodes`: for a selection of charge modes (`Off`, `Now`, `MinPV`, `PV`), including `None` which results in the param not being set
- `duration`: for duration values (e.g., `5m`, `1h30m`, `10s`)
- `float`: for floating point numbers
- `int`: for integer values
- `list`: for a list of strings (newline-separated in textarea), e.g., used for defining a list of `identifiers` for vehicles

### `choice`

`choice` defines the list of possible values when `type: choice` is used. The user can select one value from this list via a dropdown.

**Format**: Array of strings

**Example**:

```yaml
- name: schema
  type: choice
  choice: ["https", "http"]
  default: https

- name: channel
  type: choice
  choice: ["general", "feedIn", "controlledLoad"]
  required: true
```

### `advanced`

`advanced: true` marks a parameter as advanced. Advanced parameters are hidden by default in the UI and can be expanded by the user. Mostly used for non-required params that are meant for users with advanced needs and knowledge.

### `help`

`help` expects language specific help texts via `generic` (language independent), `de`, `en`

### `service`

`service` specifies an API endpoint that provides dynamic data or suggestions for this parameter during configuration. When set, the UI will call this service to provide auto-completion or pre-populated options to the user.

**Format**: `service-name/endpoint` or `service-name/endpoint?param1={param1}&param2={param2}`

Parameters from other params can be referenced using `{param-name}` syntax, which will be replaced with the user's input for that parameter. The endpoint will only be called once the user has entered values for all referenced parameters. The endpoint is called every time a referenced parameter value changes.

**UI behaviour**:

Service endpoints must return an array of strings (e.g., `["value1", "value2"]`). These values are shown as suggestions, not strict selections - users can always enter custom text values. The UI handles service responses differently based on the parameter configuration and response content:

- **Auto-fill (prepopulation)**: If the service returns exactly **one** value, the parameter is **required**, and the field is currently **empty**, the value will be automatically filled into the field.

- **Dropdown suggestions**: In all other cases (multiple values, non-required parameter, or field already has a value), the returned values are shown as a dropdown/datalist for the user to select from or ignore.

- **Empty response**: If the service returns an empty array or no data, the field remains a regular text input.

**Available services**:

- **Hardware**

  `hardware/serial`: Lists available serial ports on the system

- **Modbus**

  `modbus/read?...`: Reads a value from a modbus register (for validation/testing)

  The `modbus` service supports a special `{modbus}` parameter that will be automatically expanded to the appropriate connection parameters based on the user's modbus configuration:

  ```yaml
  # Template definition
  - name: voltage
    service: modbus/read?address=100&type=holding&{modbus}
  # Expanded for TCP connection:
  # modbus/read?address=100&type=holding&uri=192.168.1.10:502&id=1

  # Expanded for RTU connection:
  # modbus/read?address=100&type=holding&device=/dev/ttyUSB0&baudrate=9600&id=1
  ```

- **Home Assistant**

  `homeassistant/instances`: Auto-discovers Home Assistant instances on the network

  `homeassistant/entities?uri={uri}&domain=sensor`: Lists entities from a Home Assistant instance filtered by domain(s). Multiple domains can be comma-separated (e.g., `domain=sensor,binary_sensor` or `domain=number,input_number`)

## `render`

`render` contains the internal device configuration. All `param` `name` values can be used as a template variable, e.g. `{{ .host }}` for a param named `host`. The content is a go template, so all of go template feature can be used, e.g. `{{- if ... }}` statements, etc.
