package solcast

import (
	"time"

	"github.com/dylanmei/iso8601"
)

type Forecasts struct {
	Forecasts []Forecast
}

type Forecast struct {
	PvEstimate   float64   `json:"pv_estimate"`
	PvEstimate10 float64   `json:"pv_estimate10"`
	PvEstimate90 float64   `json:"pv_estimate90"`
	PeriodEnd    time.Time `json:"period_end"`
	Period       Duration
}

type Duration time.Duration

func (d *Duration) Duration() time.Duration {
	return time.Duration(*d)
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	val, err := iso8601.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = Duration(val)
	return nil
}
