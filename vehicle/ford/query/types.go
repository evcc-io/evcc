package query

type VehiclesResponse struct {
	UserVehicles struct {
		VehicleDetails []struct {
			VIN string
		}
	}
}
