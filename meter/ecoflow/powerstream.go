package ecoflow

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type PowerStream struct {
	*Device
	dataG util.Cacheable[PowerStreamData]
}

var _ api.Meter = (*PowerStream)(nil)

func NewPowerStreamFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	device, err := NewDevice(other, "ecoflow-powerstream")
	if err != nil {
		return nil, err
	}

	ps := &PowerStream{
		Device: device,
		dataG:  util.ResettableCached(func() (PowerStreamData, error) {
			var res response[PowerStreamData]
			if err := device.GetJSON(device.quotaURL(), &res); err != nil {
				return PowerStreamData{}, err
			}
			if res.Code != "0" {
				return PowerStreamData{}, fmt.Errorf("api error: %s: %s", res.Code, res.Message)
			}
			return res.Data, nil
		}, device.cache),
	}

	if device.usage == "battery" {
		return &PowerStreamBattery{ps}, nil
	}
	return ps, nil
}

func (d *PowerStream) CurrentPower() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	switch d.usage {
	case "pv":
		return data.Pv1InputWatts + data.Pv2InputWatts, nil
	case "grid":
		return data.InvOutputWatts, nil
	case "battery":
		return -data.BatWatts, nil
	default:
		return 0, fmt.Errorf("invalid usage: %s", d.usage)
	}
}

type PowerStreamBattery struct{ *PowerStream }

var _ api.Battery = (*PowerStreamBattery)(nil)

func (d *PowerStreamBattery) Soc() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}
	return float64(data.BatSoc), nil
}