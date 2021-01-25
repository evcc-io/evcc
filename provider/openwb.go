package provider

import (
	"errors"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

// OpenWBStatusProvider implements status conversion from openWB to api.Status
type OpenWBStatusProvider struct {
	plugged, charging func() (bool, error)
}

type openwbConfig = struct {
	Plugged  Config `validate:"required" ui:"de=Status Verbunden (true/false)"`
	Charging Config `validate:"required" ui:"de=Status Laden (true/false)"`
}

func init() {
	registry.Add("openwb", "OpenWB Status", NewOpenWBStatusProviderFromConfig, openwbConfig{})
}

// NewOpenWBStatusProviderFromConfig creates OpenWBStatus from given configuration
func NewOpenWBStatusProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
	var cc openwbConfig

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	plugged, err := NewBoolGetterFromConfig(cc.Plugged)

	var charging func() (bool, error)
	if err == nil {
		charging, err = NewBoolGetterFromConfig(cc.Charging)
	}

	if err != nil {
		return nil, err
	}

	o := NewOpenWBStatusProvider(plugged, charging)

	return o, nil
}

// NewOpenWBStatusProvider creates provider for OpenWB status converted from MQTT topics
func NewOpenWBStatusProvider(plugged, charging func() (bool, error)) *OpenWBStatusProvider {
	return &OpenWBStatusProvider{
		plugged:  plugged,
		charging: charging,
	}
}

// IntGetter fullfills the required IntProvider interface
// TODO replace with Go sum types
func (o *OpenWBStatusProvider) IntGetter() func() (int64, error) {
	return func() (int64, error) {
		return 0, errors.New("openWB: int provider not supported")
	}
}

// StringGetter returns string from OpenWB charging/ plugged status
func (o *OpenWBStatusProvider) StringGetter() func() (string, error) {
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
	}
}
