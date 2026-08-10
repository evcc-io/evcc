package api

import "time"

type SolarMinSocState string

const (
	SolarMinSocLow    SolarMinSocState = "low"
	SolarMinSocMedium SolarMinSocState = "medium"
	SolarMinSocHigh   SolarMinSocState = "high"
)

type SolarMinSocVehicleConfig struct {
	Low    int `json:"low"`
	Medium int `json:"medium"`
	High   int `json:"high"`
}

type SolarMinSocConfig struct {
	Enabled         bool                                `json:"enabled"`
	LowThreshold    float64                             `json:"lowThreshold"`
	MediumThreshold float64                             `json:"mediumThreshold"`
	Vehicles        map[string]SolarMinSocVehicleConfig `json:"vehicles"`
}

type SolarMinSocStatus struct {
	SolarMinSocConfig
	AvailableVehicles []SolarMinSocVehicle `json:"availableVehicles"`
	Available         bool                 `json:"available"`
	ForecastEnergy    float64              `json:"forecastEnergy"`
	State             SolarMinSocState     `json:"state,omitempty"`
	Updated           time.Time            `json:"updated,omitempty"`
}

type SolarMinSocVehicle struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}
