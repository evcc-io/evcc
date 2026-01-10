package ecoflow

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// PowerStream represents an EcoFlow PowerStream Micro-Inverter (Inverter + Battery)
type PowerStream struct {
	*Device
	dataG util.Cacheable[PowerStreamData]
}

var _ api.Meter = (*PowerStream)(nil)

// NewPowerStreamFromConfig creates EcoFlow PowerStream meter from config
func NewPowerStreamFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	cc := &Config{}
	if err := cc.Decode(other, "ecoflow-powerstream"); err != nil {
		return nil, err
	}

	device, err := NewPowerStream(cc.URI, cc.SN, cc.AccessKey, cc.SecretKey, cc.Usage, cc.Cache)
	if err != nil {
		return nil, err
	}

	// For battery usage, wrap in battery interface
	if cc.Usage == UsageBattery {
		return &PowerStreamBattery{device}, nil
	}

	// For other usages (pv, grid), return as meter
	return device, nil
}

// NewPowerStream creates an EcoFlow PowerStream device for use as a meter
func NewPowerStream(uri, sn, accessKey, secretKey string, usage Usage, cache time.Duration) (*PowerStream, error) {
	device, err := NewDevice("ecoflow-powerstream", uri, sn, accessKey, secretKey, usage, cache)
	if err != nil {
		return nil, err
	}

	ps := &PowerStream{
		Device: device,
	}

	// Create cached data fetcher
	ps.dataG = util.ResettableCached(ps.getQuotaAll, cache)

	return ps, nil
}

// getQuotaAll fetches device quota data from API
func (d *PowerStream) getQuotaAll() (PowerStreamData, error) {
	uri := fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", d.GetURI(), d.GetSN())

	var res ecoflowResponse[PowerStreamData]
	if err := d.GetJSON(uri, &res); err != nil {
		return PowerStreamData{}, err
	}

	if res.Code != "0" {
		return PowerStreamData{}, fmt.Errorf("API error:  %s: %s", res.Code, res.Message)
	}

	return res.Data, nil
}

// CurrentPower implements the api.Meter interface
func (d *PowerStream) CurrentPower() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	switch d.GetUsage() {
	case UsagePV:
		// PV power is sum of both strings
		return data.Pv1InputWatts + data.Pv2InputWatts, nil
	case UsageGrid:
		// Grid power (calculated from AC output)
		return data.InvOutputWatts, nil
	case UsageBattery:
		// Battery: negative = charging, positive = discharging
		// EcoFlow convention: BatWatts positive=discharge, negative=charge
		// evcc convention: negative=discharge, positive=charge
		return -data.BatWatts, nil
	default:
		return 0, fmt.Errorf("unknown usage type: %s", d.GetUsage())
	}
}

// PowerStreamBattery wraps PowerStream for battery interface
type PowerStreamBattery struct {
	*PowerStream
}

var _ api.Battery = (*PowerStreamBattery)(nil)

// Soc implements the api.Battery interface
func (d *PowerStreamBattery) Soc() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	return float64(data.BatSoc), nil
}
