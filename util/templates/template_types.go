package templates

import (
	"bufio"
	"fmt"
	"reflect"
	"strings"
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

const (
	ParamValueTypeString      = "string"
	ParamValueTypeNumber      = "number"
	ParamValueTypeFloat       = "float"
	ParamValueTypeBool        = "bool"
	ParamValueTypeStringList  = "stringlist"
	ParamValueTypeChargeModes = "chargemodes"
	ParamValueTypeDuration    = "duration"
)

var ValidParamValueTypes = []string{ParamValueTypeString, ParamValueTypeNumber, ParamValueTypeFloat, ParamValueTypeBool, ParamValueTypeStringList, ParamValueTypeChargeModes, ParamValueTypeDuration}

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
	Generic string // language independent
	DE      string // german text
	EN      string // english text
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
// Generelle Reihenfolge der Werte (außer Description, Default, ValueType):
// 1. defaults.yaml param section
// 2. defaults.yaml presets
// 3. defaults.yaml modbus section
// 4. template
type Param struct {
	Reference     bool         // if this is references another param definition
	Referencename string       // name of the referenced param if it is not identical to the defined name
	Preset        string       // Reference a predefined se of params
	Name          string       // Param name which is used for assigning defaults properties and referencing in render
	Description   TextLanguage // language specific titles (presented in UI instead of Name)
	Required      bool         // cli if the user has to provide a non empty value
	Mask          bool         // cli if the value should be masked, e.g. for passwords
	Advanced      bool         // cli if the user does not need to be asked. Requires a "Default" to be defined.
	Hidden        bool         // cli if the parameter should not be presented in the cli, the default value be assigned
	Deprecated    bool         // if the parameter is deprecated and thus should not be presented in the cli or docs
	Default       string       // default value if no user value is provided in the configuration
	Example       string       // cli example value
	Help          TextLanguage // cli configuration help
	Value         string       // user provided value via cli configuration
	Values        []string     // user provided list of values e.g. for ValueType "stringlist"
	ValueType     string       // string representation of the value type, "string" is default
	ValidValues   []string     // list of valid values the user can provide
	Choice        []string     // defines a set of choices, e.g. "grid", "pv", "battery", "charge" for "usage"
	Requirements  Requirements // requirements for this param to be usable, only supported via ValueType "bool"

	Baudrate int    // device specific default for modbus RS485 baudrate
	Comset   string // device specific default for modbus RS485 comset
	Port     int    // device specific default for modbus TCPIP port
	ID       int    // device specific default for modbus ID
}

// return a default value or example value depending on the renderMode
func (p *Param) DefaultValue(renderMode string) interface{} {
	// return empty list to allow iterating over in template
	if p.ValueType == ParamValueTypeStringList {
		return []string{}
	}

	if renderMode == TemplateRenderModeDocs && p.Default == "" {
		return p.Example
	}

	return p.Default
}

// overwrite specific properties by using values from another param
//
// always overwrites if not provided empty: description, valuetype, default, mask, required
//
// only overwrite if not provided empty and empty in param: help, example, requirements
func (p *Param) OverwriteProperties(withParam Param) {
	// always overwrite if defined
	p.Description.Update(withParam.Description, true)

	if withParam.ValueType != "" {
		p.ValueType = withParam.ValueType
	}

	if withParam.Required {
		p.Required = withParam.Required
	}

	if withParam.Default != "" {
		p.Default = withParam.Default
	}

	if withParam.Mask {
		p.Mask = withParam.Mask
	}

	// only set if empty
	p.Help.Update(withParam.Help, false)

	if p.Example == "" && withParam.Example != "" {
		p.Example = withParam.Example
	}

	if p.ValidValues == nil && withParam.ValidValues != nil {
		p.ValidValues = withParam.ValidValues
	}

	if reflect.DeepEqual(p.Requirements, Requirements{}) {
		p.Requirements = withParam.Requirements
	}
}

// Product contains naming information about a product a template supports
type Product struct {
	Brand       string       // product brand
	Description TextLanguage // product name
}

func (p Product) Title(lang string) string {
	return strings.TrimSpace(fmt.Sprintf("%s %s", p.Brand, p.Description.String(lang)))
}

// TemplateDefinition contains properties of a device template
type TemplateDefinition struct {
	Template     string
	Group        string    // the group this template belongs to, references groupList entries
	Covers       []string  // list of covered outdated template names
	Products     []Product // list of products this template is compatible with
	Capabilities []string
	Requirements Requirements
	Linked       []LinkedTemplate // a list of templates that should be processed as part of the guided setup
	Params       []Param
	Render       string // rendering template
}
