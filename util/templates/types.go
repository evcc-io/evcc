package templates

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/imdario/mergo"
)

const (
	ParamUsage  = "usage"
	ParamModbus = "modbus"

	UsageChoiceGrid    = "grid"
	UsageChoicePV      = "pv"
	UsageChoiceBattery = "battery"
	UsageChoiceCharge  = "charge"

	HemsTypeSMA = "sma"

	ModbusChoiceRS485    = "rs485"
	ModbusChoiceTCPIP    = "tcpip"
	ModbusKeyRS485Serial = "rs485serial"
	ModbusKeyRS485TCPIP  = "rs485tcpip"
	ModbusKeyTCPIP       = "tcpip"

	ModbusParamNameId       = "id"
	ModbusParamNameDevice   = "device"
	ModbusParamNameBaudrate = "baudrate"
	ModbusParamNameComset   = "comset"
	ModbusParamNameURI      = "uri"
	ModbusParamNameHost     = "host"
	ModbusParamNamePort     = "port"
	ModbusParamNameRTU      = "rtu"

	TemplateRenderModeDocs     = "docs"
	TemplateRenderModeUnitTest = "unittest"
	TemplateRenderModeInstance = "instance"
)

var (
	ValidModbusChoices = []string{ModbusChoiceRS485, ModbusChoiceTCPIP}
	ValidUsageChoices  = []string{UsageChoiceGrid, UsageChoicePV, UsageChoiceBattery, UsageChoiceCharge}
)

const (
	DependencyCheckEmpty    = "empty"
	DependencyCheckNotEmpty = "notempty"
	DependencyCheckEqual    = "equal"
)

var ValidDependencies = []string{DependencyCheckEmpty, DependencyCheckNotEmpty, DependencyCheckEqual}

const (
	CapabilityISO151182 = "iso151182" // ISO 15118-2 support
	CapabilityMilliAmps = "mA"        // Granular current control support
	CapabilityRFID      = "rfid"      // RFID support
	Capability1p3p      = "1p3p"      // 1P/3P phase switching support
	CapabilitySMAHems   = "smahems"   // SMA HEMS Support
)

var ValidCapabilities = []string{CapabilityISO151182, CapabilityMilliAmps, CapabilityRFID, Capability1p3p, CapabilitySMAHems}

const (
	RequirementEEBUS       = "eebus"       // EEBUS Setup is required
	RequirementMQTT        = "mqtt"        // MQTT Setup is required
	RequirementSponsorship = "sponsorship" // Sponsorship is required
)

var ValidRequirements = []string{RequirementEEBUS, RequirementMQTT, RequirementSponsorship}

var predefinedTemplateProperties = []string{
	"type", "template", "name",
	ModbusParamNameId, ModbusParamNameDevice, ModbusParamNameBaudrate, ModbusParamNameComset,
	ModbusParamNameURI, ModbusParamNameHost, ModbusParamNamePort, ModbusParamNameRTU,
	ModbusKeyTCPIP, ModbusKeyRS485Serial, ModbusKeyRS485TCPIP,
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
func (t *TextLanguage) MarshalJSON() (out []byte, err error) {
	mu.Lock()
	s := t.String(encoderLanguage)
	mu.Unlock()
	return json.Marshal(s)
}

// Requirements
type Requirements struct {
	EVCC        []string     // EVCC requirements, e.g. sponsorship
	Description TextLanguage // Description of requirements, e.g. how the device needs to be prepared
	URI         string       // URI to a webpage with more details about the preparation requirements
}

// Linked Template
type LinkedTemplate struct {
	Template        string
	Usage           string // usage: "grid", "pv", "battery"
	Multiple        bool   // if true, multiple instances of this template can be added
	ExcludeTemplate string // only consider this if no device of the named linked template was added
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
	Name          string       // Param name which is used for assigning defaults properties and referencing in render
	Description   TextLanguage // language specific titles (presented in UI instead of Name)
	Help          TextLanguage // cli configuration help
	Reference     *bool        `json:",omitempty"` // if this is references another param definition
	ReferenceName string       `json:",omitempty"` // name of the referenced param if it is not identical to the defined name
	Preset        string       `json:"-"`          // Reference a predefined set of params
	Required      *bool        `json:",omitempty"` // cli if the user has to provide a non empty value
	Mask          *bool        `json:",omitempty"` // cli if the value should be masked, e.g. for passwords
	Advanced      *bool        `json:",omitempty"` // cli if the user does not need to be asked. Requires a "Default" to be defined.
	Deprecated    *bool        `json:",omitempty"` // if the parameter is deprecated and thus should not be presented in the cli or docs
	Default       string       `json:",omitempty"` // default value if no user value is provided in the configuration
	Example       string       `json:",omitempty"` // cli example value
	Value         string       `json:"-"`          // user provided value via cli configuration
	Values        []string     `json:",omitempty"` // user provided list of values e.g. for Type "stringlist"
	Usages        []string     `json:",omitempty"` // restrict param to these usage types, e.g. "battery" for home battery capacity
	Type          ParamType    // string representation of the value type, "string" is default
	ValidValues   []string     `json:",omitempty"` // list of valid values the user can provide
	Choice        []string     `json:",omitempty"` // defines a set of choices, e.g. "grid", "pv", "battery", "charge" for "usage"
	AllInOne      *bool        `json:"-"`          // defines if the defined usages can all be present in a single device
	Requirements  Requirements `json:"-"`          // requirements for this param to be usable, only supported via Type "bool"

	// TODO move somewhere else should not be part of the param definition
	Baudrate int    `json:"-"` // device specific default for modbus RS485 baudrate
	Comset   string `json:"-"` // device specific default for modbus RS485 comset
	Port     int    `json:"-"` // device specific default for modbus TCPIP port
	ID       int    `json:"-"` // device specific default for modbus ID
}

// DefaultValue returns a default or example value depending on the renderMode
func (p *Param) DefaultValue(renderMode string) interface{} {
	// return empty list to allow iterating over in template
	if p.Type == TypeStringList {
		return []string{}
	}

	if (renderMode == TemplateRenderModeDocs || renderMode == TemplateRenderModeUnitTest) && p.Default == "" {
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

func (p *Param) IsReference() bool {
	return p.Reference != nil && *p.Reference
}

func (p *Param) IsAdvanced() bool {
	return p.Advanced != nil && *p.Advanced
}

func (p *Param) IsMask() bool {
	return p.Mask != nil && *p.Mask
}

func (p *Param) IsRequired() bool {
	return p.Required != nil && *p.Required
}

func (p *Param) IsDeprecated() bool {
	return p.Deprecated != nil && *p.Deprecated
}

func (p *Param) IsAllInOne() bool {
	return p.AllInOne != nil && *p.AllInOne
}

// Product contains naming information about a product a template supports
type Product struct {
	Brand       string       // product brand
	Description TextLanguage `json:",omitempty"` // product name
}

func (p Product) Title(lang string) string {
	return strings.TrimSpace(fmt.Sprintf("%s %s", p.Brand, p.Description.String(lang)))
}

// TemplateDefinition contains properties of a device template
type TemplateDefinition struct {
	Template     string
	Group        string           `json:",omitempty"` // the group this template belongs to, references groupList entries
	Covers       []string         `json:",omitempty"` // list of covered outdated template names
	Products     []Product        `json:",omitempty"` // list of products this template is compatible with
	Capabilities []string         `json:",omitempty"`
	Requirements Requirements     `json:"-"`
	Linked       []LinkedTemplate `json:",omitempty"` // a list of templates that should be processed as part of the guided setup
	Params       []Param          `json:",omitempty"`
	Render       string           `json:"-"` // rendering template
}
