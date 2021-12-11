package psa

import (
	"fmt"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

const (
	MQTT_SERVER      = "ssl://mwa.mpsa.com:8885"
	MQTT_RESP_TOPIC  = "psa/RemoteServices/to/cid"
	MQTT_EVENT_TOPIC = "psa/RemoteServices/events/MPHRTServices"
	MQTT_TOKEN_TTL   = 890
)

type Mqtt struct {
	realm  string
	id     string
	vin    string
	client *mqtt.Client
}

// NewMqtt creates a new vehicle
func NewMqtt(log *util.Logger, identity oauth2.TokenSource, realm, id, vin string) (*Mqtt, error) {
	client, err := mqtt.NewClient(log, "", "", "", "", 1, func(o *paho.ClientOptions) {
		o.AddBroker(MQTT_SERVER)
	})
	if err != nil {
		return nil, err
	}

	v := &Mqtt{
		realm:  realm,
		id:     id,
		vin:    vin,
		client: client,
	}

	v.client.Listen(fmt.Sprintf("%s/%s", MQTT_EVENT_TOPIC, vin), v.onMessage)
	return v, nil
}

func (v *Mqtt) onMessage(payload string) {
	fmt.Println("onMessage", string(payload))
}
