package psa

import (
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

type Mqtt struct {
	realm  string
	id     string
	client *mqtt.Client
}

// NewMqtt creates a new vehicle
func NewMqtt(log *util.Logger, identity oauth2.TokenSource, realm, id string) (*Mqtt, error) {
	client, err := mqtt.NewClient(log, "", "", "", "", 1, func(o *paho.ClientOptions) {
		o.AddBroker(MQTT_SERVER)
	})
	if err != nil {
		return nil, err
	}

	v := &Mqtt{
		realm:  realm,
		id:     id,
		client: client,
	}

	return v, nil
}
