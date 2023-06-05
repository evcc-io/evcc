package provider

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/provider/sma"
	"github.com/evcc-io/evcc/util"
	"gitlab.com/bboehmke/sunny"
)

// SMA provider
type SMA struct {
	device *sma.Device
	value  sunny.ValueID
	scale  float64
}

func init() {
	registry.Add("sma", NewSMAFromConfig)
}

// NewSMAFromConfig creates SMA provider
func NewSMAFromConfig(other map[string]interface{}) (Provider, error) {
	cc := struct {
		URI, Password, Interface string
		Serial                   uint32
		Value                    string
		Scale                    float64
	}{
		Password: "0000",
		Scale:    1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	discoverer, err := sma.GetDiscoverer(cc.Interface)
	if err != nil {
		return nil, fmt.Errorf("failed to get discoverer failed: %w", err)
	}

	provider := &SMA{
		scale: cc.Scale,
	}
	switch {
	case cc.URI != "":
		provider.device, err = discoverer.DeviceByIP(cc.URI, cc.Password)
		if err != nil {
			return nil, err
		}

	case cc.Serial > 0:
		provider.device = discoverer.DeviceBySerial(cc.Serial, cc.Password)
		if provider.device == nil {
			return nil, fmt.Errorf("device not found: %d", cc.Serial)
		}

	default:
		return nil, errors.New("missing uri or serial")
	}

	provider.value, err = sunny.ValueIDString(cc.Value)
	if err != nil {
		return nil, err
	}

	return provider, err
}

// FloatGetter creates handler for float64
func (p *SMA) FloatGetter() func() (float64, error) {
	return func() (float64, error) {
		values, err := p.device.Values()
		if err != nil {
			return 0, err
		}

		return sma.AsFloat(values[p.value]) * p.scale, nil
	}
}

// IntGetter creates handler for int64
func (p *SMA) IntGetter() func() (int64, error) {
	fl := p.FloatGetter()

	return func() (int64, error) {
		f, err := fl()
		if err != nil {
			return 0, err
		}

		return int64(f), nil
	}
}
