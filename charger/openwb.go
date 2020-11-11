package charger

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger/openwb"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

func init() {
	registry.Add("openwb", NewOpenWBFromConfig)
}

// OpenWB configures generic charger and charge meter for an openWB loadpoint
type OpenWB struct {
	api.Charger
	api.Meter
}

// NewOpenWBFromConfig creates openWB charger from config
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

	return NewOpenWB(cc.Broker, cc.User, cc.Password, cc.Topic, cc.ID, cc.Timeout)
}

// NewOpenWB creates openWB charger with given MQTT configuration
func NewOpenWB(broker, user, password, topic string, id int, timeout time.Duration) (*OpenWB, error) {
	clientID := provider.MqttClientID()
	client := provider.NewMqttClient(broker, user, password, clientID, 1)

	// check if loadpoint configured
	configured := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ConfiguredTopic), timeout)
	if isConfigured, err := configured(); err != nil || !isConfigured {
		return nil, fmt.Errorf("openWB loadpoint %d is not configured", id)
	}

	// adapt plugged/charging to status
	plugged := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.PluggedTopic), timeout)
	charging := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ChargingTopic), timeout)
	status := provider.NewOpenWBStatusProvider(plugged, charging).StringGetter

	// remaining getters
	enabled := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.EnabledTopic), timeout)

	// setters
	enable := client.BoolSetter("enable", fmt.Sprintf("%s/set/lp%d/%s", topic, id, openwb.EnabledTopic), "")
	maxcurrent := client.IntSetter("maxcurrent", fmt.Sprintf("%s/set/lp%d/%s", topic, id, openwb.MaxCurrentTopic), "")

	// meter getters
	power := client.FloatGetter(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ChargePowerTopic), 1, timeout)
	totalEnergy := client.FloatGetter(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ChargeTotalEnergyTopic), 1, timeout)

	var currents []func() (float64, error)
	for i := 1; i <= 3; i++ {
		current := client.FloatGetter(fmt.Sprintf("%s/lp/%d/%s%d", topic, id, openwb.CurrentTopic, i), 1, timeout)
		currents = append(currents, current)
	}

	c, err := NewConfigurable(status, enabled, enable, maxcurrent)
	if err != nil {
		return nil, err
	}

	m, err := meter.NewConfigurable(power)
	if err != nil {
		return nil, err
	}

	res := &OpenWB{
		Charger: c,
		Meter:   m.Decorate(totalEnergy, currents, nil),
	}

	return res, nil
}
