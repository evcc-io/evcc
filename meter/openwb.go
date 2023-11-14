package meter

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openwb"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("openwb", NewOpenWBFromConfig)
}

// NewOpenWBFromConfig creates a new configurable meter
func NewOpenWBFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		mqtt.Config `mapstructure:",squash"`
		Topic       string
		Timeout     time.Duration
		Usage       string
		capacity    `mapstructure:",squash"`
	}{
		Topic:   "openWB",
		Timeout: 15 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("openwb")

	client, err := mqtt.RegisteredClientOrDefault(log, cc.Config)
	if err != nil {
		return nil, err
	}

	// timeout handler
	h, err := provider.NewMqtt(log, client, fmt.Sprintf("%s/system/%s", cc.Topic, openwb.TimestampTopic), cc.Timeout).StringGetter()
	if err != nil {
		return nil, err
	}
	to := provider.NewTimeoutHandler(h)

	boolG := func(topic string) (func() (bool, error), error) {
		g, err := provider.NewMqtt(log, client, topic, 0).BoolGetter()
		if err != nil {
			return nil, err
		}
		return to.BoolGetter(g), nil
	}

	floatG := func(topic string) (func() (float64, error), error) {
		g, err := provider.NewMqtt(log, client, topic, 0).FloatGetter()
		if err != nil {
			return nil, err
		}
		return to.FloatGetter(g), nil
	}

	var power func() (float64, error)
	var currents func() (float64, float64, float64, error)
	var soc func() (float64, error)
	var capacity func() float64

	switch strings.ToLower(cc.Usage) {
	case "grid":
		power, err = floatG(fmt.Sprintf("%s/evu/%s", cc.Topic, openwb.PowerTopic))
		if err != nil {
			return nil, err
		}

		var curr []func() (float64, error)
		for i := 1; i <= 3; i++ {
			current, err := floatG(fmt.Sprintf("%s/evu/%s%d", cc.Topic, openwb.CurrentTopic, i))
			if err != nil {
				return nil, err
			}
			curr = append(curr, current)
		}

		currents = collectPhaseProviders(curr)

	case "pv":
		configuredG, err := boolG(fmt.Sprintf("%s/pv/1/%s", cc.Topic, openwb.PvConfigured)) // first pv
		if err != nil {
			return nil, err
		}
		configured, err := configuredG()
		if err != nil {
			return nil, err
		}

		if !configured {
			return nil, errors.New("pv not available")
		}

		g, err := floatG(fmt.Sprintf("%s/pv/%s", cc.Topic, openwb.PowerTopic))
		if err != nil {
			return nil, err
		}
		power = func() (float64, error) {
			f, err := g()
			return -f, err
		}

	case "battery":
		configuredG, err := boolG(fmt.Sprintf("%s/housebattery/%s", cc.Topic, openwb.BatteryConfigured))
		if err != nil {
			return nil, err
		}
		configured, err := configuredG()
		if err != nil {
			return nil, err
		}

		if !configured {
			return nil, errors.New("battery not available")
		}

		inner, err := floatG(fmt.Sprintf("%s/housebattery/%s", cc.Topic, openwb.PowerTopic))
		if err != nil {
			return nil, err
		}
		power = func() (float64, error) {
			f, err := inner()
			return -f, err
		}

		soc, err = floatG(fmt.Sprintf("%s/housebattery/%s", cc.Topic, openwb.SocTopic))
		if err != nil {
			return nil, err
		}

		capacity = cc.capacity.Decorator()

	default:
		return nil, fmt.Errorf("invalid usage: %s", cc.Usage)
	}

	m, err := NewConfigurable(power)
	if err != nil {
		return nil, err
	}

	res := m.Decorate(nil, currents, nil, nil, soc, capacity, nil)

	return res, nil
}
