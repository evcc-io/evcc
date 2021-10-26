package configure

import (
	_ "embed"
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

const (
	UsageChoiceGrid    = "grid"
	UsageChoicePV      = "pv"
	UsageChoiceBattery = "battery"
	UsageChoiceCharge  = "charge"
)

var ValidUsageChoices = []string{UsageChoiceGrid, UsageChoicePV, UsageChoiceBattery, UsageChoiceCharge}
var UsageChoiceDescriptions = map[string]string{
	UsageChoiceGrid:    "Grid Meter",
	UsageChoicePV:      "PV Meter",
	UsageChoiceBattery: "Battery Meter",
	UsageChoiceCharge:  "Charge Meter",
}

var ValidModbusChoices = []string{ModbusChoiceRS485, ModbusChoiceTCPIP}

const (
	DeviceClassCharger = "charger"
	DeviceClassMeter   = "meter"
	DeviceClassVehicle = "vehicle"
)

const (
	DeviceCategoryCharger      = "wallbox"
	DeviceCategoryGridMeter    = "grid"
	DeviceCategoryPVMeter      = "pv"
	DeviceCategoryBatteryMeter = "battery"
	DeviceCategoryVehicle      = "vehicle"
)

const (
	defaultNameCharger      = "wallbox"
	defaultNameGridMeter    = "grid"
	defaultNamePVMeter      = "pv"
	defaultNameBatteryMeter = "battery"
	defaultNameVehicle      = "ev"
)

const (
	defaultTitleLoadpoint = "Garage"
	defaultTitleSite      = "My Home"
)

type DeviceCategoryData struct {
	title, class, usageFilter, defaultName string
}

var DeviceCategories map[string]DeviceCategoryData = map[string]DeviceCategoryData{
	DeviceCategoryCharger:      {title: "wallbox", class: DeviceClassCharger, defaultName: defaultNameCharger},
	DeviceCategoryGridMeter:    {title: "grid meter", class: DeviceClassMeter, usageFilter: UsageChoiceGrid, defaultName: defaultNameGridMeter},
	DeviceCategoryPVMeter:      {title: "pv meter", class: DeviceClassMeter, usageFilter: UsageChoicePV, defaultName: defaultNamePVMeter},
	DeviceCategoryBatteryMeter: {title: "battery meter", class: DeviceClassMeter, usageFilter: UsageChoiceBattery, defaultName: defaultNameBatteryMeter},
	DeviceCategoryVehicle:      {title: "vehicle", class: DeviceClassVehicle, defaultName: defaultNameVehicle},
}

const itemNotPresent string = "My item is not in this list"

var ErrItemNotPresent = errors.New("item not present")

//go:embed configure.tpl
var configTmpl string

type device struct {
	Name            string
	Yaml            string
	ChargerHasMeter bool // only used with chargers to detect if we need to ask for a charge meter
}

type loadpoint struct {
	Title    string
	Charger  string
	Meter    string
	Vehicles []string
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
	EEBUS string
}
