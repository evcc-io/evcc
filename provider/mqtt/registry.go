package mqtt

import "fmt"

type registry map[string]*Client

func (r registry) Add(name string, client *Client) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate plugin type: %s", name))
	}
	r[name] = client
}

func (r registry) Get(name string) (*Client, error) {
	client, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("plugin type not registered: %s", name)
	}
	return client, nil
}

// Registry is the Mqtt client registry
var Registry registry = make(map[string]*Client)
