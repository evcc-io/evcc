package configure

const (
	DefaultConfigFilename string = "evcc.yaml"
)

type UsageChoice string

func (u UsageChoice) String() string {
	return string(u)
}

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
	class                      DeviceClass
	categoryFilter             DeviceCategory
	defaultName                string
}

var DeviceCategories = map[DeviceCategory]DeviceCategoryData{
	DeviceCategoryCharger: {
		class:       DeviceClassCharger,
		defaultName: defaultNameCharger,
	},
	DeviceCategoryGuidedSetup: {
		class: DeviceClassMeter,
	},
	DeviceCategoryGridMeter: {
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryGridMeter,
		defaultName:    defaultNameGridMeter,
	},
	DeviceCategoryPVMeter: {
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryPVMeter,
		defaultName:    defaultNamePVMeter,
	},
	DeviceCategoryBatteryMeter: {
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryBatteryMeter,
		defaultName:    defaultNameBatteryMeter,
	},
	DeviceCategoryVehicle: {
		class:       DeviceClassVehicle,
		defaultName: defaultNameVehicle,
	},
	DeviceCategoryChargeMeter: {
		class:          DeviceClassMeter,
		categoryFilter: DeviceCategoryChargeMeter,
		defaultName:    defaultNameChargeMeter,
	},
}

type localizeMap map[string]interface{}
