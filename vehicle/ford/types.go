package ford

import (
	"strings"
	"time"
)

type VehiclesResponse struct {
	UserVehicles struct {
		VehicleDetails []struct {
			VIN string
		}
	}
}

// StatusResponse is the response to the vehicle status request
type StatusResponse struct {
	VehicleStatus struct {
		BatteryFillLevel struct {
			Value     float64
			Timestamp Timestamp
		}
		ElVehDTE struct {
			Value     float64
			Timestamp Timestamp
		}
		ChargingStatus struct {
			Value     string
			Timestamp Timestamp
		}
		ChargeEndTime struct {
			Value     Timestamp
			Timestamp Timestamp
		}
		PlugStatus struct {
			Value     int
			Timestamp Timestamp
		}
		Odometer struct {
			Value     float64
			Timestamp Timestamp
		}
		Gps struct {
			Latitude  float64 `json:",string"`
			Longitude float64 `json:",string"`
			GpsState  string
			Timestamp Timestamp
		}
		LastRefresh Timestamp
	}
	Status int
}

const TimeFormat = "01-02-2006 15:04:05" // time format used by Ford API, time is in UTC

// Timestamp implements JSON unmarshal
type Timestamp struct {
	time.Time
}

// UnmarshalJSON decodes string timestamp into time.Time
func (ct *Timestamp) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")

	t, err := time.Parse(TimeFormat, s)
	if err == nil {
		(*ct).Time = t
	}

	return err
}
