package configure

import (
	"errors"
)

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
var UsageChoiceDescriptions = map[DeviceCategory]string{
	DeviceCategoryGridMeter:    "Grid Meter",
	DeviceCategoryPVMeter:      "PV Meter",
	DeviceCategoryBatteryMeter: "Battery Meter",
	DeviceCategoryChargeMeter:  "Charge Meter",
}

const (
	defaultNameCharger      = "wallbox"
	defaultNameGridMeter    = "grid"
	defaultNamePVMeter      = "pv"
	defaultNameBatteryMeter = "battery"
	defaultNameChargeMeter  = "charge"
	defaultNameVehicle      = "ev"
)

const (
	defaultTitleLoadpoint = "Garage"
	defaultTitleSite      = "Mein Zuhause"
)

type DeviceCategoryData struct {
	title, article string
	class          DeviceClass
	categoryFilter DeviceCategory
	defaultName    string
}

var DeviceCategories map[DeviceCategory]DeviceCategoryData = map[DeviceCategory]DeviceCategoryData{
	DeviceCategoryCharger: {
		title:       "Wallbox",
		article:     "eine",
		class:       DeviceClassCharger,
		defaultName: defaultNameCharger},
	DeviceCategoryGuidedSetup: {
		title:   "PV System",
		article: "ein",
		class:   DeviceClassMeter},
	DeviceCategoryGridMeter: {
		title:          "Netz-Stromzähler",
		article:        "einen",
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryGridMeter,
		defaultName:    defaultNameGridMeter},
	DeviceCategoryPVMeter: {
		title:          "PV Wechselrichter (oder entsprechenden Stromzähler)",
		article:        "einen",
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryPVMeter,
		defaultName:    defaultNamePVMeter},
	DeviceCategoryBatteryMeter: {
		title:          "Battery Wechselrichter (oder entsprechenden Stromzähler)",
		article:        "einen",
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryBatteryMeter,
		defaultName:    defaultNameBatteryMeter},
	DeviceCategoryVehicle: {
		title:       "Fahrzeug",
		article:     "ein",
		class:       DeviceClassVehicle,
		defaultName: defaultNameVehicle},
	DeviceCategoryChargeMeter: {
		title:          "Ladestromzähler",
		article:        "einen",
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryChargeMeter,
		defaultName:    defaultNameChargeMeter},
}

const itemNotPresent string = "Mein Gerät ist nicht in der Liste"

var ErrItemNotPresent = errors.New("Gerät nicht vorhanden")
var ErrDeviceNotValid = errors.New("Das Gerät funktioniert nicht")

var addedDeviceIndex int = 0

type device struct {
	Name            string
	Title           string
	Yaml            string
	ChargerHasMeter bool // only used with chargers to detect if we need to ask for a charge meter
}

type loadpoint struct {
	Title       string
	Charger     string
	ChargeMeter string
	Vehicles    []string
	Mode        string
	MinCurrent  int
	MaxCurrent  int
	Phases      int
}

type config struct {
	Meters     []device
	Chargers   []device
	Vehicles   []device
	Loadpoints []loadpoint
	Site       struct {
		Title     string
		Grid      string
		PVs       []string
		Batteries []string
	}
	EEBUS        string
	SponsorToken string
}
