package mqtt

import (
	"fmt"

	"github.com/andig/evcc/util"
)

type clientRegistry map[string]*Client

func (r clientRegistry) Add(name string, client *Client) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate plugin type: %s", name))
	}
	r[name] = client
}

func (r clientRegistry) Get(name string) (*Client, error) {
	client, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("plugin type not registered: %s", name)
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
