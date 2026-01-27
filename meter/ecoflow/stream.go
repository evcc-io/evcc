package ecoflow

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Stream struct {
	*Device
	dataG util.Cacheable[StreamData]
}

var _ api.Meter = (*Stream)(nil)

func NewStreamFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	device, err := NewDevice(other, "ecoflow-stream")
	if err != nil {
		return nil, err
	}

	s := &Stream{
		Device: device,
		dataG: util.ResettableCached(func() (StreamData, error) {
			var res response[StreamData]
			if err := device.GetJSON(device.quotaURL(), &res); err != nil {
				return StreamData{}, err
			}
			if res.Code != "0" {
				return StreamData{}, fmt.Errorf("api error: %s: %s", res.Code, res.Message)
			}
			return res.Data, nil
		}, device.cache),
	}

	if device.usage == "battery" {
		return &StreamBattery{s}, nil
	}
	return s, nil
}

func (s *Stream) CurrentPower() (float64, error) {
	data, err := s.dataG.Get()
	if err != nil {
		return 0, err
	}

	switch s.usage {
	case "pv":
		return data.PowGetPvSum, nil
	case "grid":
		return data.PowGetSysGrid, nil
	case "battery":
		return -data.PowGetBpCms, nil
	default:
		return 0, fmt.Errorf("invalid usage: %s", s.usage)
	}
}

type StreamBattery struct{ *Stream }

var _ api.Battery = (*StreamBattery)(nil)

// Soc implements the api.Battery interface
func (s *StreamBattery) Soc() (float64, error) {
	data, err := s.dataG.Get()
	if err != nil {
		return 0, err
	}
	return data.CmsBattSoc, nil
}
