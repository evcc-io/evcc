package charger

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger/openwb"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/provider/mqtt"
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
		mqtt.Config `mapstructure:",squash"`
		Topic       string
		ID          int
		Timeout     time.Duration
	}{
		Topic:   "openWB",
		ID:      1,
		Timeout: 15 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("openwb")

	clientID := mqtt.ClientID()
	client, err := mqtt.RegisteredClient(log, cc.Broker, cc.User, cc.Password, clientID, 1)
	if err != nil {
		return nil, err
	}

	// timeout handler
	timer := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/system/%s", cc.Topic, openwb.TimestampTopic), "", 1, cc.Timeout,
	).IntGetter()

	// getters
	boolG := func(topic string) func() (bool, error) {
		g := provider.NewMqtt(log, client, topic, "", 1, 0).BoolGetter()
		return func() (val bool, err error) {
			if val, err = g(); err == nil {
				_, err = timer()
			}
			return val, err
		}
	}

	floatG := func(topic string) func() (float64, error) {
		g := provider.NewMqtt(log, client, topic, "", 1, 0).FloatGetter()
		return func() (val float64, err error) {
			if val, err = g(); err == nil {
				_, err = timer()
			}
			return val, err
		}
	}

	// check if loadpoint configured
	configured := boolG(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.ConfiguredTopic))
	if isConfigured, err := configured(); err != nil || !isConfigured {
		return nil, fmt.Errorf("openWB loadpoint %d is not configured", cc.ID)
	}

	// adapt plugged/charging to status
	plugged := boolG(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.PluggedTopic))
	charging := boolG(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.ChargingTopic))
	status := provider.NewOpenWBStatusProvider(plugged, charging).StringGetter

	// remaining getters
	enabled := boolG(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.EnabledTopic))

	// setters
	enable := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/set/lp%d/%s", cc.Topic, cc.ID, openwb.EnabledTopic),
		"", 1, cc.Timeout).BoolSetter("enable")
	maxcurrent := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/set/lp%d/%s", cc.Topic, cc.ID, openwb.MaxCurrentTopic),
		"", 1, cc.Timeout).IntSetter("maxcurrent")

	// meter getters
	power := floatG(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.ChargePowerTopic))
	totalEnergy := floatG(fmt.Sprintf("%s/lp/%d/%s", cc.Topic, cc.ID, openwb.ChargeTotalEnergyTopic))

	var currents []func() (float64, error)
	for i := 1; i <= 3; i++ {
		current := floatG(fmt.Sprintf("%s/lp/%d/%s%d", cc.Topic, cc.ID, openwb.CurrentTopic, i))
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
