package plugin

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx("combined", NewCombinedFromConfig)
	registry.AddCtx("openwb", NewCombinedFromConfig)
}

// combinedPlugin implements status conversion from openWB to api.Status
type combinedPlugin struct {
	plugged, charging func() (bool, error)
}

// NewCombinedFromConfig creates combined provider
func NewCombinedFromConfig(ctx context.Context, other map[string]interface{}) (Plugin, error) {
	var cc struct {
		Plugged, Charging Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	plugged, err := cc.Plugged.BoolGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("plugged: %w", err)
	}

	charging, err := cc.Charging.BoolGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("charging: %w", err)
	}

	o := NewCombinedPlugin(plugged, charging)

	return o, nil
}

// NewCombinedPlugin creates provider for OpenWB status converted from MQTT topics
func NewCombinedPlugin(plugged, charging func() (bool, error)) *combinedPlugin {
	return &combinedPlugin{
		plugged:  plugged,
		charging: charging,
	}
}

var _ StringGetter = (*combinedPlugin)(nil)

// StringGetter returns string from OpenWB charging/ plugged status
func (o *combinedPlugin) StringGetter() (func() (string, error), error) {
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
