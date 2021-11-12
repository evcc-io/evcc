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
	DeviceCategorySingleSetup  = "single"
	DeviceCategoryGridMeter    = "grid"
	DeviceCategoryPVMeter      = "pv"
	DeviceCategoryBatteryMeter = "battery"
	DeviceCategoryChargeMeter  = "charge"
	DeviceCategoryVehicle      = "vehicle"
)

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
	title, article, class, usageFilter, defaultName string
}

var DeviceCategories map[string]DeviceCategoryData = map[string]DeviceCategoryData{
	DeviceCategoryCharger:      {title: "Wallbox", article: "eine", class: DeviceClassCharger, defaultName: defaultNameCharger},
	DeviceCategorySingleSetup:  {title: "Komplettsystem", article: "ein", class: DeviceClassMeter},
	DeviceCategoryGridMeter:    {title: "Netz-Stromzähler", article: "einen", class: DeviceClassMeter, usageFilter: UsageChoiceGrid, defaultName: defaultNameGridMeter},
	DeviceCategoryPVMeter:      {title: "PV Wechselrichter oder Stromzähler", article: "einen", class: DeviceClassMeter, usageFilter: UsageChoicePV, defaultName: defaultNamePVMeter},
	DeviceCategoryBatteryMeter: {title: "Battery Wechselrichter oder Stromzähler", article: "einen", class: DeviceClassMeter, usageFilter: UsageChoiceBattery, defaultName: defaultNameBatteryMeter},
	DeviceCategoryVehicle:      {title: "Fahrzeug", article: "ein", class: DeviceClassVehicle, defaultName: defaultNameVehicle},
	DeviceCategoryChargeMeter:  {title: "Ladestromzähler", article: "einen", class: DeviceClassMeter, usageFilter: UsageChoiceCharge, defaultName: defaultNameChargeMeter},
}

const itemNotPresent string = "Mein Gerät ist nicht in der Liste"

var ErrItemNotPresent = errors.New("Gerät nicht vorhanden")
var ErrDeviceNotValid = errors.New("Das Gerät funktioniert nicht")

//go:embed configure.tpl
var configTmpl string

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
