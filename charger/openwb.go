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

	log := util.NewLogger("openwb")

	clientID := provider.MqttClientID()
	client, err := provider.NewMqttClient(log, cc.Broker, cc.User, cc.Password, clientID, 1)
	if err != nil {
		return nil, err
	}

	// check if loadpoint configured
	configured := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.ConfiguredTopic), cc.Timeout)
	if isConfigured, err := configured(); err != nil || !isConfigured {
		return nil, fmt.Errorf("openWB loadpoint %d is not configured", cc.ID)
	}

	// adapt plugged/charging to status
	plugged := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.PluggedTopic), cc.Timeout)
	charging := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.ChargingTopic), cc.Timeout)
	status := provider.NewOpenWBStatusProvider(plugged, charging).StringGetter

	// remaining getters
	enabled := client.BoolGetter(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.EnabledTopic), cc.Timeout)

	// setters
	enable := client.BoolSetter("enable", fmt.Sprintf("%s/set/lp%d/%s", cc.Topic, cc.ID, openwb.EnabledTopic), "")
	maxcurrent := client.IntSetter("maxcurrent", fmt.Sprintf("%s/set/lp%d/%s", cc.Topic, cc.ID, openwb.MaxCurrentTopic), "")

	// meter getters
	power := client.FloatGetter(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.ChargePowerTopic), 1, cc.Timeout)
	totalEnergy := client.FloatGetter(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.ChargeTotalEnergyTopic), 1, cc.Timeout)

	var currents []func() (float64, error)
	for i := 1; i <= 3; i++ {
		current := client.FloatGetter(fmt.Sprintf("%s/lp/%d/%s%d", cc.Topic, cc.ID, openwb.CurrentTopic, i), 1, cc.Timeout)
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
