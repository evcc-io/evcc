package kamereon

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

// Response structure for kamereon api
type Response struct {
	Data data `json:"data"`
}

type data struct {
	Type, ID   string     // battery refresh
	Attributes attributes `json:"attributes"`
}

type attributes struct {
	Timestamp          string  `json:"timestamp"`
	ChargingStatus     float32 `json:"chargingStatus"`
	InstantaneousPower int     `json:"instantaneousPower"`
	RangeHvacOff       int     `json:"rangeHvacOff"`    // Nissan
	BatteryAutonomy    int     `json:"batteryAutonomy"` // Renault
	BatteryLevel       int     `json:"batteryLevel"`
	BatteryCapacity    int     `json:"batteryCapacity"` // Nissan
	BatteryTemperature int     `json:"batteryTemperature"`
	PlugStatus         int     `json:"plugStatus"`
	LastUpdateTime     string  `json:"lastUpdateTime"`
	ChargePower        int     `json:"chargePower"`
	RemainingTime      *int    `json:"chargingRemainingTime"`
}

// Provider is a kamereon provider
type Provider struct {
	apiG func() (interface{}, error)
}

// NewProvider returns a kamereon provider
func NewProvider(respG func() (Response, error), cache time.Duration) *Provider {
	return &Provider{
		apiG: provider.NewCached(func() (interface{}, error) {
			return respG()
		}, cache).InterfaceGetter(),
	}
}

var _ api.Battery = (*Provider)(nil)

// SoC implements the api.Vehicle interface
func (v *Provider) SoC() (float64, error) {
	res, err := v.apiG()

	if res, ok := res.(Response); err == nil && ok {
		return float64(res.Data.Attributes.BatteryLevel), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.apiG()
	if res, ok := res.(Response); err == nil && ok {
		if res.Data.Attributes.PlugStatus > 0 {
			status = api.StatusB
		}
		if res.Data.Attributes.ChargingStatus > 1.0 {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	res, err := v.apiG()

	if res, ok := res.(Response); err == nil && ok {
		return int64(res.Data.Attributes.RangeHvacOff), nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.apiG()

	if res, ok := res.(Response); err == nil && ok {
		timestamp, err := time.Parse(time.RFC3339, res.Data.Attributes.Timestamp)

		if res.Data.Attributes.RemainingTime == nil {
			return time.Time{}, api.ErrNotAvailable
		}

		return timestamp.Add(time.Duration(*res.Data.Attributes.RemainingTime) * time.Minute), err
	}

	return time.Time{}, err
}
