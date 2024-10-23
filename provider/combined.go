package provider

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx("combined", NewCombinedFromConfig)
	registry.AddCtx("openwb", NewCombinedFromConfig)
}

// combinedProvider implements status conversion from openWB to api.Status
type combinedProvider struct {
	plugged, charging func() (bool, error)
}

// NewCombinedFromConfig creates combined provider
func NewCombinedFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	var cc struct {
		Plugged, Charging Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	plugged, err := NewBoolGetterFromConfig(ctx, cc.Plugged)
	if err != nil {
		return nil, err
	}

	charging, err := NewBoolGetterFromConfig(ctx, cc.Charging)
	if err != nil {
		return nil, err
	}

	o := NewCombinedProvider(plugged, charging)

	return o, nil
}

// NewCombinedProvider creates provider for OpenWB status converted from MQTT topics
func NewCombinedProvider(plugged, charging func() (bool, error)) *combinedProvider {
	return &combinedProvider{
		plugged:  plugged,
		charging: charging,
	}
}

var _ StringProvider = (*combinedProvider)(nil)

// StringGetter returns string from OpenWB charging/ plugged status
func (o *combinedProvider) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		charging, err := o.charging()
		if err != nil {
			return "", err
		}
		if charging {
			return string(api.StatusC), nil
		}

		plugged, err := o.plugged()
		if err != nil {
			return "", err
		}
		if plugged {
			return string(api.StatusB), nil
		}

		return string(api.StatusA), nil
	}, nil
}
