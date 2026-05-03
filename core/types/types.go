package types

import (
	"time"

	"github.com/evcc-io/evcc/api"
)

// Measurement is the device measurements struct
type Measurement struct {
	Title         string    `json:"title,omitempty"`
	Icon          string    `json:"icon,omitempty"`
	Power         float64   `json:"power"`
	Energy        float64   `json:"energy,omitempty"`
	Powers        []float64 `json:"powers,omitempty"`
	Currents      []float64 `json:"currents,omitempty"`
	ExcessDCPower float64   `json:"excessdcpower,omitempty"`
	Capacity      *float64  `json:"capacity,omitempty"`
	Soc           *float64  `json:"soc,omitempty"`
	Controllable  *bool     `json:"controllable,omitempty"`
}

type BatteryForecast struct {
	Highest *BatteryForecastPoint `json:"highest,omitempty"`
	Lowest  *BatteryForecastPoint `json:"lowest,omitempty"`
}

// BatteryForecastPoint describes an extreme SOC point in the battery forecast.
// Limit indicates whether the configured SMax (for Highest) or SMin (for Lowest)
// boundary was reached, i.e. the battery becomes fully charged or empty.
type BatteryForecastPoint struct {
	Soc   float64   `json:"soc"`
	Time  time.Time `json:"time"`
	Limit bool      `json:"limit,omitempty"`
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
