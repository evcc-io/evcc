package skoda

// VehiclesResponse is the /v2/garage/vehicles api
type VehiclesResponse []Vehicle

type Vehicle struct {
	ID, VIN       string
	LastUpdatedAt string
	Specification struct {
		Title, Brand, Model string
		Battery             struct {
			CapacityInKWh int
		}
	}
	// Connectivities
	// Capabilities
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
