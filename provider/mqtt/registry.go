package mqtt

import (
	"fmt"

	"github.com/andig/evcc/util"
)

type clientRegistry map[string]*Client

func (r clientRegistry) Add(broker string, client *Client) {
	if _, exists := r[broker]; exists {
		panic(fmt.Sprintf("cannot register duplicate broker: %s", broker))
	}
	r[broker] = client
}

func (r clientRegistry) Get(broker string) (*Client, error) {
	client, exists := r[broker]
	if !exists {
		return nil, fmt.Errorf("missing mqtt broker configuration: %s", broker)
	}
	return client, nil
}

// registry is the Mqtt client registry
var registry clientRegistry = make(map[string]*Client)

// RegisteredClient reuses an registered Mqtt publisher or creates a new one
func RegisteredClient(log *util.Logger, broker, user, password, clientID string, qos byte) (*Client, error) {
	client, err := registry.Get(broker)

	if err != nil {
		if client, err = NewClient(log, broker, user, password, ClientID(), qos); err == nil {
			registry.Add(broker, client)
		}
	}

	return client, err
}
