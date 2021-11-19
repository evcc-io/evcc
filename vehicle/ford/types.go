package ford

type VehiclesResponse struct {
	Vehicles struct {
		Values []struct {
			VIN string
		} `json:"$values"`
	}
}

// VehicleStatus holds the relevant data extracted from JSON that the server sends
// on vehicle status request
type VehicleStatus struct {
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
