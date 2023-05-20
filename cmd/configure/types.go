package configure

import "github.com/evcc-io/evcc/util/config"

const (
	DefaultConfigFilename string = "evcc.yaml"
)

type UsageChoice string

func (u UsageChoice) String() string {
	return string(u)
}

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

const (
	defaultNameCharger      = "wallbox"
	defaultNameGridMeter    = "grid"
	defaultNamePVMeter      = "pv"
	defaultNameBatteryMeter = "battery"
	defaultNameChargeMeter  = "charge"
	defaultNameVehicle      = "ev"
)

type DeviceCategoryData struct {
	title, article, additional string
	class                      config.Class
	categoryFilter             DeviceCategory
	defaultName                string
}

var DeviceCategories = map[DeviceCategory]DeviceCategoryData{
	DeviceCategoryCharger: {
		class:       config.Charger,
		defaultName: defaultNameCharger,
	},
	DeviceCategoryGuidedSetup: {
		class: config.Meter,
	},
	DeviceCategoryGridMeter: {
		class:          config.Meter,
		categoryFilter: DeviceCategoryGridMeter,
		defaultName:    defaultNameGridMeter,
	},
	DeviceCategoryPVMeter: {
		class:          config.Meter,
		categoryFilter: DeviceCategoryPVMeter,
		defaultName:    defaultNamePVMeter,
	},
	DeviceCategoryBatteryMeter: {
		class:          config.Meter,
		categoryFilter: DeviceCategoryBatteryMeter,
		defaultName:    defaultNameBatteryMeter,
	},
	DeviceCategoryVehicle: {
		class:       config.Vehicle,
		defaultName: defaultNameVehicle,
	},
	DeviceCategoryChargeMeter: {
		class:          config.Meter,
		categoryFilter: DeviceCategoryChargeMeter,
		defaultName:    defaultNameChargeMeter,
	},
}

type localizeMap map[string]interface{}
