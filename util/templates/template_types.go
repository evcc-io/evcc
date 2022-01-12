package templates

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

	ModbusRS485Serial = "modbusrs485serial"
	ModbusRS485TCPIP  = "modbusrs485tcpip"
	ModbusTCPIP       = "modbustcpip"

	ModbusParamNameId        = "id"
	ModbusParamValueId       = 1
	ModbusParamNameDevice    = "device"
	ModbusParamValueDevice   = "/dev/ttyUSB0"
	ModbusParamNameBaudrate  = "baudrate"
	ModbusParamValueBaudrate = 9600
	ModbusParamNameComset    = "comset"
	ModbusParamValueComset   = "8N1"
	ModbusParamNameURI       = "uri"
	ModbusParamNameHost      = "host"
	ModbusParamValueHost     = "192.0.2.2"
	ModbusParamNamePort      = "port"
	ModbusParamValuePort     = 502
	ModbusParamNameRTU       = "rtu"

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
)

var ValidParamValueTypes = []string{ParamValueTypeString, ParamValueTypeNumber, ParamValueTypeFloat, ParamValueTypeBool, ParamValueTypeStringList, ParamValueTypeChargeModes}

var ValidModbusChoices = []string{ModbusChoiceRS485, ModbusChoiceTCPIP}
var ValidUsageChoices = []string{UsageChoiceGrid, UsageChoicePV, UsageChoiceBattery, UsageChoiceCharge}

const (
	DependencyCheckEmpty    = "empty"
	DependencyCheckNotEmpty = "notempty"
	DependencyCheckEqual    = "equal"
)

var ValidDependencies = []string{DependencyCheckEmpty, DependencyCheckNotEmpty, DependencyCheckEqual}

const (
	CapabilityISO151182 = "iso151182" // ISO 15118-2 support
	CapabilityRFID      = "rfid"      // RFID support
	Capability1p3p      = "1p3p"      // 1P/3P phase switching support
	CapabilitySMAHems   = "smahems"   // SMA HEMS Support
)

var ValidCapabilities = []string{CapabilityISO151182, CapabilityRFID, Capability1p3p, CapabilitySMAHems}

const (
	RequirementEEBUS       = "eebus"       // EEBUS Setup is required
	RequirementMQTT        = "mqtt"        // MQTT Setup is required
	RequirementSponsorship = "sponsorship" // Sponsorship is required
)

var ValidRequirements = []string{RequirementEEBUS, RequirementMQTT, RequirementSponsorship}

var predefinedTemplateProperties = []string{"type", "template", "name",
	ModbusParamNameId, ModbusParamNameDevice, ModbusParamNameBaudrate, ModbusParamNameComset,
	ModbusParamNameURI, ModbusParamNameHost, ModbusParamNamePort, ModbusParamNameRTU,
	ModbusRS485Serial, ModbusRS485TCPIP, ModbusTCPIP,
	ModbusKeyTCPIP, ModbusKeyRS485Serial, ModbusKeyRS485TCPIP,
}

// language specific texts
type TextLanguage struct {
	Generic string // language independent
	DE      string // german text
	EN      string // english text
}

func (t *TextLanguage) String(lang string) string {
	if t.Generic != "" {
		return t.Generic
	}
	switch lang {
	case "de":
		return t.DE
	case "en":
		return t.EN
	}
	return t.DE
}

func (t *TextLanguage) SetString(lang, value string) {
	switch lang {
	case "de":
		t.DE = value
	case "en":
		t.EN = value
	default:
		t.DE = value
	}
}

// Update the language specific texts
// always true to always update if the new value is not empty
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
	EVCC        []string
	Description TextLanguage // Description of requirements, e.g. how the device needs to be prepared
	URI         string       // URI to a webpage with more details about the preparation requirements
}

type GuidedSetup struct {
	Enable bool             // if true, guided setup is possible
	Linked []LinkedTemplate // a list of templates that should be processed as part of the guided setup
}

// Linked Template
type LinkedTemplate struct {
	Template        string
	Usage           string // usage: "grid", "pv", "battery"
	Multiple        bool   // if true, multiple instances of this template can be added
	ExcludeTemplate string // only consider this if no device of the named linked template was added
}

type Dependency struct {
	Name  string // the Param name value this depends on
	Check string // the check to perform, valid values see const DependencyCheck...
	Value string // the string value to check against
}

// Param is a proxy template parameter
type Param struct {
	Base         string       // Reference a predefined se of params
	Name         string       // Param name which is used for assigning defaults properties and referencing in render
	Description  TextLanguage // language specific titles (presented in UI instead of Name)
	Dependencies []Dependency // List of dependencies, when this param should be presented
	Required     bool         // cli if the user has to provide a non empty value
	Mask         bool         // cli if the value should be masked, e.g. for passwords
	Advanced     bool         // cli if the user does not need to be asked. Requires a "Default" to be defined.
	Default      string       // default value if no user value is provided in the configuration
	Example      string       // cli example value
	Help         TextLanguage // cli configuration help
	Test         string       // testing default value
	Value        string       // user provided value via cli configuration
	Values       []string     // user provided list of values
	ValueType    string       // string representation of the value type, "string" is default
	Choice       []string     // defines a set of choices, e.g. "grid", "pv", "battery", "charge" for "usage"
	Baudrate     int          // device specific default for modbus RS485 baudrate
	Comset       string       // device specific default for modbus RS485 comset
	Port         int          // device specific default for modbus TCPIP port
	ID           int          // device specific default for modbus ID
}

type ParamDefault struct {
	Name        string
	Description TextLanguage
	Help        TextLanguage
	Example     string
	ValueType   string
}

type ParamDefaultList struct {
	Params []ParamDefault
}

// return the param with the given name
func (p *ParamDefaultList) ParamByName(name string) (int, ParamDefault) {
	for i, param := range p.Params {
		if param.Name == name {
			return i, param
		}
	}
	return -1, ParamDefault{}
}

var paramDefaultList ParamDefaultList

type ParamBase struct {
	Params []Param
	Render string
}

var paramBaseList map[string]ParamBase

var groupList map[string]TextLanguage

type Product struct {
	Brand       string       // product brand
	Description TextLanguage // product name
}

type TemplateDefinition struct {
	Template     string
	Products     []Product // list of products this template is compatible with
	Capabilities []string
	Requirements Requirements
	GuidedSetup  GuidedSetup
	Group        string // the group this template belongs to, references groupList entries
	Params       []Param
	Render       string // rendering template
}
