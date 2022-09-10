package provider

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// OpenWBStatus implements status conversion from openWB to api.Status
type OpenWBStatus struct {
	plugged, charging func() (bool, error)
}

// NewOpenWBStatusProviderFromConfig creates OpenWBStatus from given configuration
func NewOpenWBStatusProviderFromConfig(other map[string]interface{}) (func() (string, error), error) {
	var cc struct {
		Plugged, Charging Config
	}
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

	return o.StringGetter, nil
}

// NewOpenWBStatusProvider creates provider for OpenWB status converted from MQTT topics
func NewOpenWBStatusProvider(plugged, charging func() (bool, error)) *OpenWBStatus {
	return &OpenWBStatus{
		plugged:  plugged,
		charging: charging,
	}
}

// StringGetter returns string from OpenWB charging/ plugged status
func (o *OpenWBStatus) StringGetter() (string, error) {
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
