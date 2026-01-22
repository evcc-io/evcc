package templates

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/gosimple/slug"
)

const (
	ParamUsage  = "usage"
	ParamModbus = "modbus"

	HemsTypeSMA = "sma"

	ModbusChoiceRS485    = "rs485"
	ModbusChoiceTCPIP    = "tcpip"
	ModbusChoiceUDP      = "udp"
	ModbusKeyRS485Serial = "rs485serial"
	ModbusKeyRS485TCPIP  = "rs485tcpip"
	ModbusKeyTCPIP       = "tcpip"
	ModbusKeyUDP         = "udp"

	ModbusParamId       = "id"
	ModbusParamDevice   = "device"
	ModbusParamBaudrate = "baudrate"
	ModbusParamComset   = "comset"
	ModbusParamURI      = "uri"
	ModbusParamHost     = "host"
	ModbusParamPort     = "port"
	ModbusParamRTU      = "rtu"
)

const (
	RenderModeDocs int = iota
	RenderModeUnitTest
	RenderModeInstance
)

var (
	ValidModbusChoices = []string{ModbusChoiceRS485, ModbusChoiceTCPIP, ModbusChoiceUDP}

	// ModbusParams contains all field names used by modbus templates
	ModbusParams = []string{
		ModbusParamId, ModbusParamDevice, ModbusParamBaudrate, ModbusParamComset,
		ModbusParamURI, ModbusParamHost, ModbusParamPort, ModbusParamRTU,
	}

	ModbusConnectionTypes = []string{
		ModbusKeyTCPIP, ModbusKeyUDP, ModbusKeyRS485Serial, ModbusKeyRS485TCPIP,
	}
)

const (
	CapabilityISO151182      = "iso151182"       // ISO 15118-2 support
	CapabilityMilliAmps      = "mA"              // Granular current control support
	CapabilityRFID           = "rfid"            // RFID support
	Capability1p3p           = "1p3p"            // 1P/3P phase switching support
	CapabilityBatteryControl = "battery-control" // Battery control support
)

var ValidCapabilities = []string{CapabilityISO151182, CapabilityMilliAmps, CapabilityRFID, Capability1p3p, CapabilityBatteryControl}

const (
	RequirementEEBUS       = "eebus"       // EEBUS Setup is required
	RequirementMQTT        = "mqtt"        // MQTT Setup is required
	RequirementSponsorship = "sponsorship" // Sponsorship is required
	RequirementSkipTest    = "skiptest"    // Template should be rendered but not tested
)

var ValidRequirements = []string{RequirementEEBUS, RequirementMQTT, RequirementSponsorship, RequirementSkipTest}

var predefinedTemplateProperties = slices.Concat(
	[]string{"type", "template", "name"}, ModbusParams, ModbusConnectionTypes,
)

// Pattern contains regex pattern and examples for input validation
type Pattern struct {
	Regex    string   `json:",omitempty"`
	Examples []string `json:",omitempty"`
}

// Validate checks if a value matches the pattern and returns a descriptive error if not
func (p *Pattern) Validate(value string) error {
	if p.Regex == "" {
		return nil
	}

	matched, err := regexp.MatchString(p.Regex, value)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	if matched {
		return nil
	}

	errMsg := fmt.Sprintf("value %q does not match required pattern", value)
	if len(p.Examples) > 0 {
		errMsg += fmt.Sprintf(". Valid examples: %s", strings.Join(p.Examples, ", "))
	}
	return errors.New(errMsg)
}

// TextLanguage contains language-specific texts
type TextLanguage struct {
	Generic string `json:",omitempty"` // language independent
	DE      string `json:",omitempty"` // german text
	EN      string `json:",omitempty"` // english text
}

func (t *TextLanguage) String(lang string) string {
	switch {
	case lang == "de" && t.DE != "":
		return t.DE
	case lang == "en" && t.EN != "":
		return t.EN
	default:
		if t.Generic != "" {
			return t.Generic
		}
		return t.DE
	}
}

// ShortString reduces help texts to one line and adds ...
func (t *TextLanguage) ShortString(lang string) string {
	help := t.String(lang)
	scanner := bufio.NewScanner(strings.NewReader(help))

	var short string
	for scanner.Scan() {
		if short == "" {
			short = scanner.Text()
		} else {
			short += "..."
			break
		}
	}

	return short
}

// Update the language specific texts
//
// always true to always update if the new value is not empty
//
// always false to update only if the old value is empty and the new value is not empty
func (t *TextLanguage) Update(new TextLanguage, always bool) {
	if (new.Generic != "" && always) || (!always && t.Generic == "" && new.Generic != "") {
		t.Generic = new.Generic
	}
	if (new.DE != "" && always) || (!always && t.DE == "" && new.DE != "") {
		t.DE = new.DE
	}
	if (new.EN != "" && always) || (!always && t.EN == "" && new.EN != "") {
		t.EN = new.EN
	}
}

// MarshalJSON implements the json.Marshaler interface
func (t TextLanguage) MarshalJSON() ([]byte, error) {
	mu.Lock()
	defer mu.Unlock()
	return json.Marshal(t.String(encoderLanguage))
}

func (r Requirements) MarshalJSON() ([]byte, error) {
	mu.Lock()
	custom := struct {
		EVCC        []string `json:",omitempty"`
		Description string   `json:",omitempty"`
	}{
		EVCC:        r.EVCC,
		Description: r.Description.String(encoderLanguage),
	}
	mu.Unlock()
	return json.Marshal(custom)
}

// Requirements
type Requirements struct {
	EVCC        []string     // EVCC requirements, e.g. sponsorship
	Description TextLanguage // Description of requirements, e.g. how the device needs to be prepared
}

// Param is a proxy template parameter
// Params can be defined:
// 1. in the template: uses entries in 4. for default properties and values, can be overwritten here
// 2. in defaults.yaml presets: can ne referenced in 1 and some values set here can be overwritten in 1. See OverwriteProperties method
// 3. in defaults.yaml modbus section: are referenced in 1 by a `name:modbus` param entry. Some values here can be overwritten in 1.
// 4. in defaults.yaml param section: defaults for some params
// Generelle Reihenfolge der Werte (außer Description, Default, Type):
// 1. defaults.yaml param section
// 2. defaults.yaml presets
// 3. defaults.yaml modbus section
// 4. template
type Param struct {
	Name        string       // Param name which is used for assigning defaults properties and referencing in render
	Description TextLanguage // language specific titles (presented in UI instead of Name)
	Help        TextLanguage // cli configuration help
	Preset      string       `json:"-"`          // Reference a predefined set of params
	Required    bool         `json:",omitempty"` // cli if the user has to provide a non empty value
	Mask        bool         `json:",omitempty"` // cli if the value should be masked, e.g. for passwords
	Private     bool         `json:",omitempty"` // value should be redacted in bug reports, e.g. email, locations, ...
	Advanced    bool         `json:",omitempty"` // cli if the user does not need to be asked. Requires a "Default" to be defined.
	Deprecated  bool         `json:",omitempty"` // if the parameter is deprecated and thus should not be presented in the cli or docs
	Default     string       `json:",omitempty"` // default value if no user value is provided in the configuration
	Example     string       `json:",omitempty"` // cli example value
	Value       string       `json:"-"`          // user provided value via cli configuration
	Values      []string     `json:",omitempty"` // user provided list of values e.g. for Type "list"
	Unit        string       `json:",omitempty"` // unit of the value, e.g. "kW", "kWh", "A", "V"
	Usages      []string     `json:",omitempty"` // restrict param to these usage types, e.g. "battery" for home battery capacity
	Type        ParamType    // string representation of the value type, "string" is default
	Choice      []string     `json:",omitempty"` // defines a set of choices, e.g. "grid", "pv", "battery", "charge" for "usage"
	Service     string       `json:",omitempty"` // defines a service to provide choices
	Pattern     *Pattern     `json:",omitempty"` // regex pattern and examples for input validation

	// TODO move somewhere else should not be part of the param definition
	Baudrate int    `json:",omitempty"` // device specific default for modbus RS485 baudrate
	Comset   string `json:",omitempty"` // device specific default for modbus RS485 comset
	Port     int    `json:",omitempty"` // device specific default for modbus TCPIP port
	ID       int    `json:",omitempty"` // device specific default for modbus ID
}

// DefaultValue returns a default or example value depending on the renderMode
func (p *Param) DefaultValue(renderMode int) any {
	// return empty list to allow iterating over in template
	if p.Type == TypeList {
		return []string{}
	}

	if (renderMode == RenderModeDocs || renderMode == RenderModeUnitTest) && p.Default == "" {
		return p.Example
	}

	return p.Default
}

// OverwriteProperties merges properties from parameter definition
func (p *Param) OverwriteProperties(withParam Param) {
	if err := mergo.Merge(p, &withParam); err != nil {
		panic(err)
	}
}

func (p *Param) IsAdvanced() bool {
	return p.Advanced && !p.Required
}

func (p *Param) IsMasked() bool {
	return p.Mask
}

func (p *Param) IsPrivate() bool {
	return p.Private
}

func (p *Param) IsRequired() bool {
	return p.Required && !p.Deprecated
}

func (p *Param) IsDeprecated() bool {
	return p.Deprecated
}

// yamlQuote quotes strings for yaml if they would otherwise by modified by the unmarshaler
func (p *Param) yamlQuote(value string) string {
	if p.Type != TypeString {
		return value
	}

	return yamlQuote(value)
}

var _ json.Marshaler = (*Param)(nil)

func (p Param) MarshalJSON() ([]byte, error) {
	type param Param
	pp := (param)(p)
	pp.Required = p.IsRequired()
	pp.Advanced = p.IsAdvanced()
	return json.Marshal(pp)
}

// Product contains naming information about a product a template supports
type Product struct {
	Brand       string       // product brand
	Description TextLanguage `json:",omitempty"` // product name
}

// Title returns the product title in the given language
func (p Product) Title(lang string) string {
	return strings.TrimSpace(fmt.Sprintf("%s %s", p.Brand, p.Description.String(lang)))
}

// Identifier returns a unique language-independent identifier for the product
func (p Product) Identifier() string {
	return slug.Make(p.Title("en"))
}

type CountryCode string

func (c CountryCode) IsValid() bool {
	// ensure ISO 3166-1 alpha-2 format
	validCode := regexp.MustCompile(`^[A-Z]{2}$`)
	return validCode.MatchString(string(c))
}
