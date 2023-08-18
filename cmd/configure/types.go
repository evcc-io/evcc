package configure

import (
	"github.com/evcc-io/evcc/util/templates"
)

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
	class                      templates.Class
	categoryFilter             DeviceCategory
	defaultName                string
}

var DeviceCategories = map[DeviceCategory]DeviceCategoryData{
	DeviceCategoryCharger: {
		class:       templates.Charger,
		defaultName: defaultNameCharger,
	},
	DeviceCategoryGuidedSetup: {
		class: templates.Meter,
	},
	DeviceCategoryGridMeter: {
		class:          templates.Meter,
		categoryFilter: DeviceCategoryGridMeter,
		defaultName:    defaultNameGridMeter,
	},
	DeviceCategoryPVMeter: {
		class:          templates.Meter,
		categoryFilter: DeviceCategoryPVMeter,
		defaultName:    defaultNamePVMeter,
	},
	DeviceCategoryBatteryMeter: {
		class:          templates.Meter,
		categoryFilter: DeviceCategoryBatteryMeter,
		defaultName:    defaultNameBatteryMeter,
	},
	DeviceCategoryVehicle: {
		class:       templates.Vehicle,
		defaultName: defaultNameVehicle,
	},
	DeviceCategoryChargeMeter: {
		class:          templates.Meter,
		categoryFilter: DeviceCategoryChargeMeter,
		defaultName:    defaultNameChargeMeter,
	},
}

type localizeMap map[string]interface{}
