package fiat

import (
	"strconv"
	"time"
)

type VehiclesResponse struct {
	Vehicles []Vehicle
}

type Vehicle struct {
	VIN string
}

type StatusResponse struct {
	VehicleInfo struct {
		Odometer struct {
			Odometer struct {
				Value int `json:",string"`
				Unit  string
			}
		}
		Timestamp TimeMillis
	}
	EvInfo struct {
		Battery struct {
			ChargingLevel   string // LEVEL_2
			ChargingStatus  string // CHARGING
			DistanceToEmpty struct {
				Value int
				Unit  string
			}
			PlugInStatus        bool // true
			StateOfCharge       int  // 75
			TimeToFullyChargeL1 int  // 0
			TimeToFullyChargeL2 int  // 540
			TotalRange          int  // 17
		}
		Timestamp TimeMillis
	}
	Timestamp TimeMillis
}

type ActionResponse struct {
	Name, Message string
}

type PinResponse struct {
	Data struct {
		Token string
	}
}

// TimeMillis implements JSON unmarshal for Unix timestamps in milliseconds
type TimeMillis struct {
	time.Time
}

// UnmarshalJSON decodes unix timestamps in ms into time.Time
func (ct *TimeMillis) UnmarshalJSON(data []byte) error {
	i, err := strconv.ParseInt(string(data), 10, 64)

	if err == nil {
		t := time.UnixMilli(i)
		(*ct).Time = t
	}

	return err
}
