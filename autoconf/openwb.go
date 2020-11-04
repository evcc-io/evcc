package autoconf

import (
	"fmt"
	"time"

	"github.com/andig/evcc/charger"
	"github.com/andig/evcc/charger/openwb"
	"github.com/andig/evcc/core"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

const timeout = 5 * time.Second

type openWBdetector struct {
	log      *util.Logger
	client   *provider.MqttClient
	broker   string
	user     string
	password string
	topic    string
}

// DetectOpenWB detects configured OpenWB loadpoints from MQTT broker
func DetectOpenWB(broker, user, password, topic string) (*core.Site, error) {
	clientID := provider.MqttClientID()
	client := provider.NewMqttClient(broker, user, password, clientID, 1)

	d := &openWBdetector{
		log:      util.NewLogger("detect"),
		client:   client,
		broker:   broker,
		user:     user,
		password: password,
		topic:    topic,
	}

	site, err := d.site()
	if err != nil {
		return nil, err
	}

	var loadpoints []*core.LoadPoint
	for id := 1; id <= 8; id++ {
		lp := d.loadpoint(id)
		if lp != nil {
			loadpoints = append(loadpoints, lp)
		}
	}

	return site, nil
}

func (d *openWBdetector) site() (*core.Site, error) {
	site := core.NewSite()

	gridG := d.client.FloatGetter(fmt.Sprintf("%s/evu/%s", d.topic, openwb.PowerTopic), 1, timeout)
	pvG := d.client.FloatGetter(fmt.Sprintf("%s/pv/%s", d.topic, openwb.PowerTopic), 1, timeout)

	grid, err := meter.NewConfigurable(gridG)
	if err != nil {
		return nil, err
	}

	pv, err := meter.NewConfigurable(pvG)
	if err != nil {
		return nil, err
	}

	_ = grid
	_ = pv

	// battery
	configuredG := d.client.BoolGetter(fmt.Sprintf("%s/housebattery/%s", d.topic, openwb.HouseBatteryConfiguredTopic), timeout)
	configured, err := configuredG()
	if err != nil {
		d.log.ERROR.Println(err)
	} else if configured {
		batteryG := d.client.FloatGetter(fmt.Sprintf("%s/housebattery/%s", d.topic, openwb.PowerTopic), 1, timeout)
		battery, err := meter.NewConfigurable(batteryG)
		if err != nil {
			return nil, err
		}

		_ = battery
	}

	return site, nil
}

func (d *openWBdetector) loadpoint(id int) *core.LoadPoint {
	lpTopic := fmt.Sprintf("%s/lp/%d/%s", d.topic, id, openwb.ConfiguredTopic)
	configuredG := d.client.BoolGetter(lpTopic, timeout)

	configured, err := configuredG()
	if err != nil {
		d.log.ERROR.Println(err)
		return nil
	}

	if configured {
		d.log.INFO.Printf("openWB: found loadpoint: %d", id)

		c, err := charger.NewOpenWB(d.broker, d.user, d.password, d.topic, id, timeout)
		if err != nil {
			d.log.ERROR.Printf("openWB: configuring loadpoint %d failed:", err)
			return nil
		}

		_ = c

		lp := core.NewLoadPoint(nil)

		_ = lp
	}

	return nil
}
