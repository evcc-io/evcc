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
	MileageInKm float64
}

// ChargerResponse is the /v2/charging/<vin> api
type ChargerResponse struct {
	IsVehicleInSaveLocation bool
	Status                  struct {
		ChargingRateInKilometersPerHour      float64
		ChargePowerInKw                      float64
		RemainingTimeToFullyChargedInMinutes int64
		State                                string
		ChargeType                           string
		Battery                              struct {
			RemainingCruisingRangeInMeters int64
			StateOfChargeInPercent         int
		}
	}
	Settings SettingsResponse
}

// SettingsResponse is the /v1/charging/<vin>/settings api
type SettingsResponse struct {
	AutoUnlockPlugWhenCharged    string `json:"autoUnlockPlugWhenCharged"`
	MaxChargeCurrentAc           string `json:"maxChargeCurrentAc"`
	TargetStateOfChargeInPercent int    `json:"targetStateOfChargeInPercent"`
}

// ChargerResponse is the /v2/air-conditioning/<vin> api
type ClimaterResponse struct {
	State                  string `json:"state"`
	ChargerConnectionState string `json:"chargerConnectionState"`
	ChargerLockState       string `json:"chargerLockState"`
}
