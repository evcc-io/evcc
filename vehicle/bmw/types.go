package bmw

type Vehicle struct {
	VIN            string
	Model          string
	AppVehicleType string
}

type VehicleStatus struct {
	StatusCode int
	Message    string
	State      struct {
		CurrentMileage        int64
		Range                 int64
		ElectricChargingState struct {
			ChargingLevelPercent int64
			Range                int64
			IsChargerConnected   bool
			ChargingStatus       string
			ChargingTarget       int64
		}
	}
}
