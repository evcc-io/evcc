package porsche

import "encoding/json"

// Vehicle is an entry from GET /connect/v1/vehicles
type Vehicle struct {
	VIN       string `json:"vin"`
	ModelName string `json:"modelName"`
	ModelType struct {
		Model  string `json:"model"`
		Year   string `json:"year"`
		Engine string `json:"engine"` // BEV, PHEV, COMBUSTION
	} `json:"modelType"`
}

// StatusResponse is the overview response for a single vehicle.
type StatusResponse struct {
	VIN          string        `json:"vin"`
	ModelName    string        `json:"modelName"`
	Measurements []Measurement `json:"measurements"`
}

// Measurement is a single keyed measurement; Value is decoded on demand.
type Measurement struct {
	Key    string `json:"key"`
	Status struct {
		IsEnabled bool   `json:"isEnabled"`
		Cause     string `json:"cause"`
	} `json:"status"`
	Value json.RawMessage `json:"value"`
}

// measurement returns the (enabled) measurement for the given key.
func (s StatusResponse) measurement(key string) (Measurement, bool) {
	for _, m := range s.Measurements {
		if m.Key == key && m.Status.IsEnabled && len(m.Value) > 0 {
			return m, true
		}
	}
	return Measurement{}, false
}

// decode unmarshals the value of the measurement with the given key into v.
// Returns false if the measurement is missing/disabled.
func (s StatusResponse) decode(key string, v any) bool {
	m, ok := s.measurement(key)
	if !ok {
		return false
	}
	return json.Unmarshal(m.Value, v) == nil
}

// measurement value shapes

type batteryLevel struct {
	Percent float64 `json:"percent"`
}

type rangeValue struct {
	Kilometers float64 `json:"kilometers"`
}

type chargingSummary struct {
	Status string `json:"status"` // e.g. CHARGING, CHARGING_COMPLETED, NOT_CHARGING, NOT_PLUGGED
	Mode   string `json:"mode"`   // e.g. DIRECT
	Type   string `json:"type"`   // e.g. AC, DC
}

type chargingRate struct {
	ChargingPower float64 `json:"chargingPower"`
	RateKph       float64 `json:"chargingRate-kph"`
}

type climatizerState struct {
	IsOn bool `json:"isOn"`
}

type gpsLocation struct {
	Location  string  `json:"location"` // "lat,lng"
	Direction float64 `json:"direction"`
}
