package pleos

import (
	"fmt"
	"strconv"

	"github.com/evcc-io/evcc/api"
)

// Vehicle is an entry of the /vehicles/consent list
type Vehicle struct {
	VIN      string `json:"vin"`
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
	Type     string `json:"type"` // EV
	Sellname string `json:"sellname"`
}

// Title returns the user-facing vehicle name
func (v Vehicle) Title() string {
	if v.Nickname != "" {
		return v.Nickname
	}
	return v.Sellname
}

// Battery is the /vehicles/{vin}/batteries payload
type Battery struct {
	Charge struct {
		Plugin              string  `json:"plugin"` // connected, disconnected, invalid
		Status              string  `json:"status"` // notCharging, charging, fastCharging, ...
		Charging            bool    `json:"charging"`
		StateOfCharge       float64 `json:"stateOfCharge"`
		TargetStateOfCharge struct {
			Standard int64 `json:"standard"`
			Quick    int64 `json:"quick"`
		} `json:"targetStateOfCharge"`
		RemainTime int64 `json:"remainTime"` // minutes
	} `json:"charge"`
	Timestamp string `json:"timestamp"`
}

// Powertrain is the /vehicles/{vin}/powertrains payload
type Powertrain struct {
	DistanceToEmpties []struct {
		Type  string `json:"type"`  // ICE, EV, HEV, PHEV, FCEV, invalid
		Value string `json:"value"` // number or "invalid"
		Unit  string `json:"unit"`
	} `json:"distanceToEmpties"`
}

// Driving is the /vehicles/{vin}/driving payload
type Driving struct {
	Odometer Distance `json:"odometer"`
}

// Status is the /vehicles/{vin}/status payload
type Status struct {
	ClimateControl struct {
		Status string `json:"status"` // on, off
	} `json:"climateControl"`
}

// Location is the /vehicles/{vin}/locations payload
type Location struct {
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
}

// Distance is a value with distance unit
type Distance struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"` // km, mile, meter, feet, invalid
}

// Km converts the distance to kilometers
func (d Distance) Km() (float64, error) {
	return km(d.Value, d.Unit)
}

// km converts value in the given api distance unit to kilometers
func km(value float64, unit string) (float64, error) {
	switch unit {
	case "km", "":
		return value, nil
	case "mile", "miles":
		return value * 1.609344, nil
	case "meter":
		return value / 1e3, nil
	case "feet":
		return value * 0.0003048, nil
	}
	return 0, fmt.Errorf("invalid distance unit: %s", unit)
}

// Range returns the electric range in km from the EV distance entry
func (p Powertrain) Range() (int64, error) {
	for _, dte := range p.DistanceToEmpties {
		if dte.Type != "EV" {
			continue
		}

		// value is "invalid" or "-" when the vehicle hasn't reported a range
		f, err := strconv.ParseFloat(dte.Value, 64)
		if err != nil {
			break
		}

		f, err = km(f, dte.Unit)
		return int64(f), err
	}

	return 0, api.ErrNotAvailable
}
