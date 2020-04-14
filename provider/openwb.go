package provider

import (
	"time"

	"github.com/andig/evcc/api"
)

// openWBConfig is the specific mqtt getter/setter configuration
type openWBConfig struct {
	PlugStat, ChargeStat string
	Timeout              time.Duration
}

type openWBStatusProvider struct {
	plugStat, chargeStat BoolGetter
}

func openWBStatusFromConfig(log *api.Logger, other map[string]interface{}) (res StringGetter) {
	if MQTT == nil {
		log.FATAL.Fatal("mqtt not configured")
	}

	var pc openWBConfig
	api.DecodeOther(log, other, &pc)

	o := &openWBStatusProvider{
		plugStat:   MQTT.BoolGetter(pc.PlugStat, pc.Timeout),
		chargeStat: MQTT.BoolGetter(pc.ChargeStat, pc.Timeout),
	}

	return o.stringGetter
}

func (o *openWBStatusProvider) stringGetter() (string, error) {
	charging, err := o.chargeStat()
	if err != nil {
		return "", err
	}
	if charging {
		return string(api.StatusC), nil
	}

	plugged, err := o.plugStat()
	if err != nil {
		return "", err
	}
	if plugged {
		return string(api.StatusB), nil
	}

	return string(api.StatusA), nil
}
