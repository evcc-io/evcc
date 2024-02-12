package ford

type VehiclesResponse struct {
	UserVehicles struct {
		VehicleDetails []struct {
			VIN string
		}
	}
}
