package connectedcars

// TokenResponse is the response from the device token login endpoint.
type TokenResponse struct {
	Token   string `json:"token"`
	Expires int    `json:"expires"`
}

// VehiclesResponse is the GraphQL response for listing vehicles.
type VehiclesResponse struct {
	Data *struct {
		Vehicles struct {
			Items []Vehicle `json:"items"`
		} `json:"vehicles"`
	} `json:"data"`
}

// Vehicle represents a vehicle from the Connected Cars API.
type Vehicle struct {
	ID           string `json:"id"`
	LicensePlate string `json:"licensePlate"`
	VIN          string `json:"vin"`
}

// DataResponse is the GraphQL response for vehicle data.
type DataResponse struct {
	Data *struct {
		Vehicle VehicleData `json:"vehicle"`
	} `json:"data"`
}

// VehicleData contains the vehicle telemetry fields.
type VehicleData struct {
	ChargePercentage *struct {
		Pct float64 `json:"pct"`
	} `json:"chargePercentage"`
	Odometer *struct {
		Odometer float64 `json:"odometer"`
	} `json:"odometer"`
	RangeTotalKm *struct {
		Km float64 `json:"km"`
	} `json:"rangeTotalKm"`
}
