package configure

const (
	ModbusMagicSetupComment  string = "# ::modbus-setup::"
	ModbusChoiceRS485        string = "rs485"
	ModbusChoiceTCPIP        string = "tcpip"
	ModbusKeyRS485Serial     string = "rs485serial"
	ModbusKeyRS485TCPIP      string = "rs485tcpip"
	ModbusKeyTCPIP           string = "tcpip"
	ModbusParamNameId        string = "id"
	ModbusParamValueId       string = "1"
	ModbusParamNameDevice    string = "device"
	ModbusParamValueDevice   string = "/dev/ttyUSB0"
	ModbusParamNameBaudrate  string = "baudrate"
	ModbusParamValueBaudrate string = "9600"
	ModbusParamNameComset    string = "comset"
	ModbusParamValueComset   string = "8N1"
	ModbusParamNameURI       string = "uri"
	ModbusParamNameHost      string = "host"
	ModbusParamValueHost     string = "192.0.2.2"
	ModbusParamNamePort      string = "port"
	ModbusParamValuePort     string = "502"
	ModbusParamNameRTU       string = "rtu"
)

type UsageChoice string

func (u UsageChoice) String() string {
	return string(u)
}

var ValidModbusChoices = []string{ModbusChoiceRS485, ModbusChoiceTCPIP}

type DeviceClass string

func (c DeviceClass) String() string {
	return string(c)
}

const (
	DeviceClassCharger DeviceClass = "charger"
	DeviceClassMeter   DeviceClass = "meter"
	DeviceClassVehicle DeviceClass = "vehicle"
)

type DeviceCategory string

func (c DeviceCategory) String() string {
	return string(c)
}

const (
	DeviceCategoryCharger      DeviceCategory = "wallbox"
	DeviceCategoryGridMeter    DeviceCategory = "grid"
	DeviceCategoryPVMeter      DeviceCategory = "pv"
	DeviceCategoryBatteryMeter DeviceCategory = "battery"
	DeviceCategoryChargeMeter  DeviceCategory = "charge"
	DeviceCategoryVehicle      DeviceCategory = "vehicle"
	DeviceCategoryGuidedSetup  DeviceCategory = "guided"
)

var ValidUsageChoices = []DeviceCategory{DeviceCategoryGridMeter, DeviceCategoryPVMeter, DeviceCategoryBatteryMeter, DeviceCategoryChargeMeter}

const (
	defaultNameCharger      = "wallbox"
	defaultNameGridMeter    = "grid"
	defaultNamePVMeter      = "pv"
	defaultNameBatteryMeter = "battery"
	defaultNameChargeMeter  = "charge"
	defaultNameVehicle      = "ev"
)

type DeviceCategoryData struct {
	title, article string
	class          DeviceClass
	categoryFilter DeviceCategory
	defaultName    string
}

var DeviceCategories map[DeviceCategory]DeviceCategoryData = map[DeviceCategory]DeviceCategoryData{
	DeviceCategoryCharger: {
		class:       DeviceClassCharger,
		defaultName: defaultNameCharger},
	DeviceCategoryGuidedSetup: {
		class: DeviceClassMeter},
	DeviceCategoryGridMeter: {
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryGridMeter,
		defaultName:    defaultNameGridMeter},
	DeviceCategoryPVMeter: {
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryPVMeter,
		defaultName:    defaultNamePVMeter},
	DeviceCategoryBatteryMeter: {
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryBatteryMeter,
		defaultName:    defaultNameBatteryMeter},
	DeviceCategoryVehicle: {
		class:       DeviceClassVehicle,
		defaultName: defaultNameVehicle},
	DeviceCategoryChargeMeter: {
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryChargeMeter,
		defaultName:    defaultNameChargeMeter},
}

type localizeMap map[string]interface{}
