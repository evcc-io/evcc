//go:build mqtt

package cmd

import (
	"fmt"
	"strings"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
)

// setup mqtt
func configureMqtt(conf *globalconfig.Mqtt) error {
	// migrate settings
	if settings.Exists(keys.Mqtt) {
		if err := settings.Json(keys.Mqtt, &conf); err != nil {
			return err
		}

		// TODO remove yaml file
		// } else {
		// 	// migrate settings & write defaults
		// 	if err := settings.SetJson(keys.Mqtt, conf); err != nil {
		// 		return err
		// 	}
	}

	if conf.Broker == "" {
		return nil
	}

	log := util.NewLogger("mqtt")

	instance, err := mqtt.RegisteredClient(log, conf.Broker, conf.User, conf.Password, conf.ClientID, 1, conf.Insecure, func(options *paho.ClientOptions) {
		topic := fmt.Sprintf("%s/status", strings.Trim(conf.Topic, "/"))
		options.SetWill(topic, "offline", 1, true)

		oc := options.OnConnect
		options.SetOnConnectHandler(func(client paho.Client) {
			oc(client)                                   // original handler
			_ = client.Publish(topic, 1, true, "online") // alive - not logged
		})
	})
	if err != nil {
		return fmt.Errorf("failed configuring mqtt: %w", err)
	}

	mqtt.Instance = instance
	return nil
}
