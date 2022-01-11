package bmw

type VehiclesResponse struct {
	Vehicles []Vehicle
}

type Vehicle struct {
	VIN   string
	Model string
}

type VehiclesStatusResponse []VehicleStatus

type VehicleStatus struct {
	VIN, Model, Brand string
	Year              int
	Properties        struct {
		ChargingState *struct {
			ChargePercentage   int
			State              string // CHARGING, ERROR, FINISHED_FULLY_CHARGED, FINISHED_NOT_FULL, INVALID, NOT_CHARGING, WAITING_FOR_CHARGING, COMPLETED
			IsChargerConnected bool
		}
		ElectricRange *struct {
			Distance struct {
				Value int
			}
		}
	}
	Status struct {
		CurrentMileage *struct {
			Mileage int
		}
	}
}
