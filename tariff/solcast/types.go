package solcast

import "time"

type Forecasts struct {
	Forecasts []Forecast
}

type Forecast struct {
	PvEstimate   float64   `json:"pv_estimate"`
	PvEstimate10 float64   `json:"pv_estimate10"`
	PvEstimate90 float64   `json:"pv_estimate90"`
	PeriodEnd    time.Time `json:"period_end"`
	Period       string
}
