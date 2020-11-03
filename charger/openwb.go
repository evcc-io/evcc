package charger

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

// OpenWB is an api.Charger implementation for an OpenWB slave
// using the default charger implementation and MQTT
type OpenWB struct{}

func init() {
	registry.Add("openwb", NewOpenWBFromConfig)
}

// predefined openWB topic names
const (
	OpenWBConfiguredTopic = "boolChargePointConfigured"
	OpenWBEnabledTopic    = "ChargePointEnabled"
	OpenWBMaxCurrentTopic = "DirectChargeAmps"
)

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
	plugged := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, "boolPlugStat"), cc.Timeout)
	charging := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, "boolChargeStat"), cc.Timeout)
	status := provider.NewOpenWBStatusProvider(plugged, charging).StringGetter

	enabled := client.BoolGetter(fmt.Sprintf("%s/%d/%s", cc.Topic, cc.ID, OpenWBEnabledTopic), cc.Timeout)

	enable := client.BoolSetter("enable", fmt.Sprintf("%s/set/lp%d/%s", cc.Topic, cc.ID, OpenWBEnabledTopic), "%enable%")
	maxcurrent := client.IntSetter("current", fmt.Sprintf("%s/set/lp%d/%s", cc.Topic, cc.ID, OpenWBMaxCurrentTopic), "%current%")

	return NewConfigurable(status, enabled, enable, maxcurrent)
}
