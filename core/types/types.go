package types

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

// Measurement is the device measurements struct
type Measurement struct {
	Title         string    `json:"title,omitempty"`
	Icon          string    `json:"icon,omitempty"`
	Power         float64   `json:"power"`                   // W
	Import        float64   `json:"import,omitempty"`        // kWh
	Powers        []float64 `json:"powers,omitempty"`        // W
	Currents      []float64 `json:"currents,omitempty"`      // A
	ExcessDCPower float64   `json:"excessdcpower,omitempty"` // W
	Capacity      *float64  `json:"capacity,omitempty"`      // kWh
	Soc           *float64  `json:"soc,omitempty"`           // %
	Controllable  *bool     `json:"controllable,omitempty"`
}

type BatteryForecast struct {
	Full  *time.Time `json:"full"`
	Empty *time.Time `json:"empty"`
}

var _ api.TitleDescriber = (*Measurement)(nil)

// GetTitle implements api.TitleDescriber interface for InfluxDB tagging
func (m Measurement) GetTitle() string {
	return m.Title
}

type BatteryState struct {
	Power    float64          `json:"power"`
	Energy   float64          `json:"energy,omitempty"`
	Capacity float64          `json:"capacity,omitempty"`
	Soc      float64          `json:"soc"`
	Devices  []Measurement    `json:"devices,omitempty" influxdb:"battery"`
	Forecast *BatteryForecast `json:"forecast,omitempty"`
}
