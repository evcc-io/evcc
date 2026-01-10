package ecoflow

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Stream represents an EcoFlow Stream Energy Management System (Inverter + Battery)
type Stream struct {
	*Device
	dataG util.Cacheable[StreamData]
}

var _ api.Meter = (*Stream)(nil)

// NewStreamFromConfig creates EcoFlow Stream meter from config
func NewStreamFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	cc := &Config{}
	if err := cc.Decode(other, "ecoflow-stream"); err != nil {
		return nil, err
	}

	device, err := NewStream(cc.URI, cc.SN, cc.AccessKey, cc.SecretKey, cc.Usage, cc.Cache)
	if err != nil {
		return nil, err
	}

	// For battery usage, wrap in battery interface
	if cc.Usage == UsageBattery {
		return &StreamBattery{device}, nil
	}

	// For other usages (pv, grid), return as meter
	return device, nil
}

// NewStream creates an EcoFlow Stream device for use as a meter
func NewStream(uri, sn, accessKey, secretKey string, usage Usage, cache time.Duration) (*Stream, error) {
	device, err := NewDevice("ecoflow-stream", uri, sn, accessKey, secretKey, usage, cache)
	if err != nil {
		return nil, err
	}

	s := &Stream{
		Device: device,
	}

	// Create cached data fetcher
	s.dataG = util.ResettableCached(s.getQuotaAll, cache)

	return s, nil
}

// getQuotaAll fetches device quota data from API
func (d *Stream) getQuotaAll() (StreamData, error) {
	uri := fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", d.GetURI(), d.GetSN())

	var res ecoflowResponse[StreamData]
	if err := d.GetJSON(uri, &res); err != nil {
		return StreamData{}, err
	}

	if res.Code != "0" {
		return StreamData{}, fmt.Errorf("API error: %s: %s", res.Code, res.Message)
	}

	return res.Data, nil
}

// CurrentPower implements the api.Meter interface
func (d *Stream) CurrentPower() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	switch d.GetUsage() {
	case UsagePV:
		return data.PowGetPvSum, nil
	case UsageGrid:
		return data.PowGetSysGrid, nil
	case UsageBattery:
		// Battery: negative = charging, positive = discharging
		// EcoFlow convention: PowGetBpCms positive when discharging, negative when charging
		// evcc convention: negative when discharging, positive when charging
		return -data.PowGetBpCms, nil
	default:
		return 0, fmt.Errorf("unknown usage type: %s", d.GetUsage())
	}
}

// StreamBattery wraps Stream for battery interface
type StreamBattery struct {
	*Stream
}

var _ api.Battery = (*StreamBattery)(nil)

// Soc implements the api.Battery interface
func (d *StreamBattery) Soc() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	return data.CmsBattSoc, nil
}
