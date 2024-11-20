package fiat

import (
	"fmt"
	"html"
	"strconv"
	"time"
)

type ErrorInfo struct {
	ErrorCode    int
	ErrorMessage string
	ErrorDetails string
}

func (e ErrorInfo) Error() error {
	if e.ErrorCode == 0 {
		return nil
	}
	return fmt.Errorf("%s: %s", e.ErrorMessage, html.UnescapeString(e.ErrorDetails))
}

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
	EvInfo *struct {
		Battery struct {
			ChargingLevel   string // LEVEL_2
			ChargingStatus  string // CHARGING
			DistanceToEmpty struct {
				Value int
				Unit  string
			}
			PlugInStatus        bool    // true
			StateOfCharge       float64 // 75
			TimeToFullyChargeL1 int     // 0
			TimeToFullyChargeL2 int     // 540
			TotalRange          int     // 17
		}
		Timestamp TimeMillis
	} `json:",omitempty"`
	Timestamp TimeMillis
}

type LocationResponse struct {
	TimeStamp        TimeMillis
	Longitude        float64
	Latitude         float64
	Altitude         float64
	Bearing          float64
	IsLocationApprox bool
}

type ActionResponse struct {
	Name, Message string

	// deep refresh
	Command          string
	CorrelationId    string
	ResponseStatus   string
	StatusTimestamp  TimeMillis
	AsyncRespTimeout int
}

type PinAuthResponse struct {
	Name, Message string
	Token         string
	Expiry        int64 // ms duration
}

// TimeMillis implements JSON unmarshal for Unix timestamps in milliseconds
type TimeMillis struct {
	time.Time
}

// UnmarshalJSON decodes unix timestamps in ms into time.Time
func (ct *TimeMillis) UnmarshalJSON(data []byte) error {
	i, err := strconv.ParseInt(string(data), 10, 64)

	if err == nil {
		t := time.Unix(0, i*1e6)
		ct.Time = t
	}

	return err
}
