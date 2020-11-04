package charger

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger/openwb"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

func init() {
	registry.Add("openwb", NewOpenWBFromConfig)
}

// NewOpenWBFromConfig creates a new configurable charger
func NewOpenWBFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Broker         string
		User, Password string
		Topic          string
		ID             int
		Timeout        time.Duration
	}{
		Topic:   "openWB",
		ID:      1,
		Timeout: 5 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	clientID := provider.MqttClientID()
	client := provider.NewMqttClient(cc.Broker, cc.User, cc.Password, clientID, 1)

	// adapt plugged/charging to status
	plugged := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.PluggedTopic), cc.Timeout)
	charging := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.ChargingTopic), cc.Timeout)
	status := provider.NewOpenWBStatusProvider(plugged, charging).StringGetter

	// remaining getters
	enabled := client.BoolGetter(fmt.Sprintf("%s/%d/%s", cc.Topic, cc.ID, openwb.EnabledTopic), cc.Timeout)

	// setters
	enable := client.BoolSetter("enable", fmt.Sprintf("%s/set/lp%d/%s", cc.Topic, cc.ID, openwb.EnabledTopic), "${enable}")
	maxcurrent := client.IntSetter("maxcurrent", fmt.Sprintf("%s/set/lp%d/%s", cc.Topic, cc.ID, openwb.MaxCurrentTopic), "${maxcurrent}")

	return NewConfigurable(status, enabled, enable, maxcurrent)
}
