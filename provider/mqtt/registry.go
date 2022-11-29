package mqtt

import (
	"errors"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/util"
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

var (
	mu       sync.Mutex
	registry clientRegistry = make(map[string]*Client)
)

// RegisteredClient reuses an registered Mqtt publisher or creates a new one
func RegisteredClient(log *util.Logger, broker, user, password, clientID string, qos byte, insecure bool, opts ...Option) (*Client, error) {
	key := fmt.Sprintf("%s.%s:%s", broker, user, password)

	mu.Lock()
	defer mu.Unlock()
	client, err := registry.Get(key)

	if err != nil {
		if clientID == "" {
			clientID = ClientID()
		}

		if client, err = NewClient(log, broker, user, password, clientID, qos, insecure, opts...); err == nil {
			registry.Add(key, client)
		}
	}

	return client, err
}

// RegisteredClientOrDefault reuses an registered Mqtt publisher or creates a new one.
// If no publisher is configured, it uses the default instance.
func RegisteredClientOrDefault(log *util.Logger, cc Config) (*Client, error) {
	var err error
	client := Instance

	if cc.Broker != "" {
		client, err = RegisteredClient(log, cc.Broker, cc.User, cc.Password, cc.ClientID, 1, cc.Insecure)
	}

	if client == nil && err == nil {
		err = errors.New("missing mqtt broker configuration")
	}

	return client, err
}
