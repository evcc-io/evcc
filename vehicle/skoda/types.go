package skoda

// VehiclesResponse is the /v3/garage api
type VehiclesResponse struct {
	Vehicles []Vehicle
}

type Vehicle struct {
	ID, VIN       string
	Name          string // user-given name
	LastUpdatedAt string
	Specification Specification
	// Connectivities
	// Capabilities
}

type Specification struct {
	Title         string
	Model         string
	ModelYear     string
	Body          string
	SystemCode    string
	SystemModelId string
	Engine        struct {
		Typ       string `json:"type"`
		PowerInKW int
	}
	Battery struct {
		CapacityInKWh int
	}
	Gearbox struct {
		Typ string `json:"type"`
	}
	TrimLevel            string
	ManufacturingDate    string
	MaxChargingPowerInKW int
}

// StatusResponse is the /v2/vehicle-status/<vin> api
type StatusResponse struct {
	Remote struct {
		MileageInKm float64
	}
}

// ChargerResponse is the /v1/charging/<vin>/status api
type ChargerResponse struct {
	Plug struct {
		ConnectionState string // Connected
		LockState       string // Unlocked
	}
	Charging struct {
		State                           string // Error
		RemainingToCompleteInSeconds    int64
		ChargingPowerInWatts            float64
		ChargingRateInKilometersPerHour float64
		ChargingType                    string // Invalid
		ChargeMode                      string // MANUAL
	}
	Battery struct {
		CruisingRangeElectricInMeters int64
		StateOfChargeInPercent        int
	}
}

// SettingsResponse is the /v1/charging/<vin>/settings api
type SettingsResponse struct {
	AutoUnlockPlugWhenCharged    string `json:"autoUnlockPlugWhenCharged"`
	MaxChargeCurrentAc           string `json:"maxChargeCurrentAc"`
	TargetStateOfChargeInPercent int    `json:"targetStateOfChargeInPercent"`
}
