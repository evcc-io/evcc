package connect

type VehiclesResponse struct {
	UserVehicles struct {
		VehicleDetails []struct {
			VIN string
		}
	}
}

type InformationResponse struct {
	Status string
}
