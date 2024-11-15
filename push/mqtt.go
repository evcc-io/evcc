package push

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("mqtt", NewMqttFromConfig)
}

// Mqtt implements the MQTT messenger
type Mqtt struct {
	log    *util.Logger
	client *mqtt.Client
	topic  string
}

// NewMqttFromConfig creates a new Mqtt messenger with the MQTT endpoing configured properly
func NewMqttFromConfig(other map[string]interface{}) (Messenger, error) {
	var cc struct {
		Topic string
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	client := mqtt.Instance
	if client == nil {
		return nil, errors.New("no mqtt broker configured")
	}
	topic := cc.Topic
	if topic == "" {
		topic = "evcc/events"
	}
	m := &Mqtt{
		log:    util.NewLogger("mqtt"),
		client: client,
		topic:  topic,
	}

	return m, nil
}

// Send sends the message via MQTT
func (m *Mqtt) Send(title, msg string) {
	jsonPayload, err := prepareJsonMessage(title, msg)
	if err != nil {
		m.log.ERROR.Println("error creating MQTT payload:", err)
		return
	}

	m.log.DEBUG.Printf("sending MQTT event to %s: %s", m.topic, string(jsonPayload))
	err = m.client.Publish(m.topic, false, jsonPayload)
	if err != nil {
		m.log.ERROR.Println("messenger mqtt publish:", err)
	}
}

func prepareJsonMessage(title string, msg string) ([]byte, error) {
	if json.Valid([]byte(msg)) {
		var jsonPayload map[string]interface{}
		// msg is JSON, parse it
		if err := json.Unmarshal([]byte(msg), &jsonPayload); err != nil {
			return nil, fmt.Errorf("json unmarshal: %w", err)
		}
		if title != "" {
			jsonPayload["title"] = title
		}
		payload, err := json.Marshal(jsonPayload)
		if err != nil {
			return nil, fmt.Errorf("json marshal: %w", err)
		}
		return payload, nil
	}

	// msg is not JSON so take it literally. Ignore title in this case
	return []byte(msg), nil
}
