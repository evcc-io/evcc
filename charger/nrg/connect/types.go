package connect

const (
	SettingsPath     = "settings"
	MeasurementsPath = "measurements"
)

// Measurements is the /api/measurements response
type Measurements struct {
	Message               string `json:"omitempty"` // api message if not ok
	ChargingEnergy        float64
	ChargingEnergyOverAll float64
	ChargingPower         float64
	ChargingPowerPhase    [3]float64
	ChargingCurrentPhase  [3]float64
	Frequency             float64
}

// Settings is the /api/settings request/response
type Settings struct {
	Message string `json:",omitempty"` // api message if not ok
	Info    Info   `json:",omitempty"`
	Values  Values
}

// Info is Settings.Info
type Info struct {
	Connected bool `json:",omitempty"`
}

// Values is Settings.Values
type Values struct {
	ChargingStatus  *ChargingStatus  `json:",omitempty"`
	ChargingCurrent *ChargingCurrent `json:",omitempty"`
	DeviceMetadata  DeviceMetadata
}

// ChargingStatus is Settings.Values.ChargingStatus
type ChargingStatus struct {
	Charging bool
}

// ChargingCurrent is Settings.Values.ChargingCurrent
type ChargingCurrent struct {
	Value float64
}

// DeviceMetadata is Settings.Values.DeviceMetadata
type DeviceMetadata struct {
	Password string
}
