package types

import (
	"encoding/json"
	"time"

	"github.com/evcc-io/evcc/api"
)

// Measurement is the device measurements struct
type Measurement struct {
	Title         string           `json:"title,omitempty"`
	Icon          string           `json:"icon,omitempty"`
	Power         float64          `json:"power"`
	Energy        float64          `json:"energy,omitempty"`
	Powers        []float64        `json:"powers,omitempty"`
	Currents      []float64        `json:"currents,omitempty"`
	ExcessDCPower float64          `json:"excessdcpower,omitempty"`
	Capacity      *float64         `json:"capacity,omitempty"`
	Soc           *float64         `json:"soc,omitempty"`
	Controllable  *bool            `json:"controllable,omitempty"`
	Forecast      *BatteryForecast `json:"forecast,omitempty"`
}

type BatteryForecast struct {
	Full  time.Time `json:"full"`
	Empty time.Time `json:"empty"`
}

func (f BatteryForecast) MarshalJSON() ([]byte, error) {
	var full, empty int64
	if !f.Full.IsZero() {
		full = int64(time.Until(f.Full).Seconds())
	}
	if !f.Empty.IsZero() {
		empty = int64(time.Until(f.Empty).Seconds())
	}

	return json.Marshal(struct {
		BatteryForecast
		UntilFull  int64 `json:"untilFull,omitempty"`
		UntilEmpty int64 `json:"untilEmpty,omitempty"`
	}{
		BatteryForecast: f,
		UntilFull:       full,
		UntilEmpty:      empty,
	})
}

var _ api.TitleDescriber = (*Measurement)(nil)

// GetTitle implements api.TitleDescriber interface for InfluxDB tagging
func (m Measurement) GetTitle() string {
	return m.Title
}

type BatteryState struct {
	Power    float64       `json:"power"`
	Energy   float64       `json:"energy,omitempty"`
	Capacity float64       `json:"capacity,omitempty"`
	Soc      float64       `json:"soc"`
	Devices  []Measurement `json:"devices,omitempty" influxdb:"battery"`
}
