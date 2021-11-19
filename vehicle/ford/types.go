package ford

type VehiclesResponse struct {
	Vehicles struct {
		Values []struct {
			VIN string
		} `json:"$values"`
	}
}

// StatusResponse is the response to the vehicle status request
type StatusResponse struct {
	VehicleStatus struct {
		BatteryFillLevel struct {
			Value     float64
			Timestamp string
		}
		ElVehDTE struct {
			Value     float64
			Timestamp string
		}
		ChargingStatus struct {
			Value     string
			Timestamp string
		}
		PlugStatus struct {
			Value     int
			Timestamp string
		}
		LastRefresh string
	}
	Status int
}
