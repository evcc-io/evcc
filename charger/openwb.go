package charger

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger/openwb"
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
	currentPowerG func() (float64, error)
	totalEnergyG  func() (float64, error)
	currentsG     []func() (float64, error)
}

// NewOpenWBFromConfig creates a new configurable charger
func NewOpenWBFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		mqtt.Config `mapstructure:",squash"`
		Topic       string
		Timeout     time.Duration
		ID          int
	}{
		Topic:   openwb.RootTopic,
		Timeout: openwb.Timeout,
		ID:      1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("openwb")

	return NewOpenWB(log, cc.Config, cc.ID, cc.Topic, cc.Timeout)
}

// NewOpenWB creates a new configurable charger
func NewOpenWB(log *util.Logger, mqttconf mqtt.Config, id int, topic string, timeout time.Duration) (*OpenWB, error) {
	client, err := mqtt.RegisteredClientOrDefault(log, mqttconf)
	if err != nil {
		return nil, err
	}

	// timeout handler
	timer := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/system/%s", topic, openwb.TimestampTopic), "", 1, timeout,
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
	configured := boolG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ConfiguredTopic))
	if isConfigured, err := configured(); err != nil || !isConfigured {
		return nil, fmt.Errorf("openWB loadpoint %d is not configured", id)
	}

	// adapt plugged/charging to status
	plugged := boolG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.PluggedTopic))
	charging := boolG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ChargingTopic))
	status := provider.NewOpenWBStatusProvider(plugged, charging).StringGetter

	// remaining getters
	enabled := boolG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.EnabledTopic))

	// setters
	enable := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/set/lp%d/%s", topic, id, openwb.EnabledTopic),
		"", 1, timeout).BoolSetter("enable")
	maxcurrent := provider.NewMqtt(log, client,
		fmt.Sprintf("%s/set/lp%d/%s", topic, id, openwb.MaxCurrentTopic),
		"", 1, timeout).IntSetter("maxcurrent")

	// meter getters
	currentPowerG := floatG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ChargePowerTopic))
	totalEnergyG := floatG(fmt.Sprintf("%s/lp/%d/%s", topic, id, openwb.ChargeTotalEnergyTopic))

	var currentsG []func() (float64, error)
	for i := 1; i <= 3; i++ {
		current := floatG(fmt.Sprintf("%s/lp/%d/%s%d", topic, id, openwb.CurrentTopic, i))
		currentsG = append(currentsG, current)
	}

	charger, err := NewConfigurable(status, enabled, enable, maxcurrent)
	if err != nil {
		return nil, err
	}

	res := &OpenWB{
		Charger:       charger,
		currentPowerG: currentPowerG,
		totalEnergyG:  totalEnergyG,
		currentsG:     currentsG,
	}

	return res, nil
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *OpenWB) CurrentPower() (float64, error) {
	return m.currentPowerG()
}

// TotalEnergy implements the Meter.TotalEnergy interface
func (m *OpenWB) TotalEnergy() (float64, error) {
	return m.totalEnergyG()
}

// Currents implements the Meter.Currents interface
func (m *OpenWB) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, currentG := range m.currentsG {
		c, err := currentG()
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, c)
	}

	return currents[0], currents[1], currents[2], nil
}
