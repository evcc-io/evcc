package autoconf

import (
	"fmt"
	"time"

	"github.com/andig/evcc/charger"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
)

const timeout = time.Second

// DetectOpenWB detects configured OpenWB loadpoints from MQTT broker
func DetectOpenWB(broker, user, password, topic string) error {
	log := util.NewLogger("detect")

	clientID := provider.MqttClientID()
	client := provider.NewMqttClient(broker, user, password, clientID, 1)

	for id := 1; id < 10; id++ {
		lp := fmt.Sprintf("%s/lp/%d/%s", topic, id, charger.OpenWBConfiguredTopic)
		configuredG := client.BoolGetter(lp, timeout)

		configured, err := configuredG()
		if err != nil {
			log.ERROR.Println(err)
			continue
		}

		if configured {
			log.INFO.Printf("openWB: found loadpoint: %d", id)

			if err := createOpenWBLoadpoint(broker, user, password, topic, id); err != nil {
				log.ERROR.Printf("openWB: configuring loadpoint %d failed:", err)
			}
		}
	}

	return nil
}

func createOpenWBLoadpoint(broker, user, password, topic string, id int) error {
	c, err := charger.NewOpenWBFromConfig(map[string]interface{}{
		"broker":   broker,
		"user":     user,
		"password": password,
		"topic":    topic,
		"id":       id,
	})
	_ = c

	return err
}
