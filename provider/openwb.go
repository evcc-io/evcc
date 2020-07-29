package provider

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

type openWBStatusProvider struct {
	plugged, charging func() (bool, error)
}

func openWBStatusFromConfig(other map[string]interface{}) (func() (string, error), error) {
	cc := struct {
		Plugged, Charging Config
	}{}
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

	o := &openWBStatusProvider{
		plugged:  plugged,
		charging: charging,
	}

	return o.stringGetter, nil
}

func (o *openWBStatusProvider) stringGetter() (string, error) {
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
