package homeassistant

import (
	"fmt"
	"sync"

	"golang.org/x/oauth2"
)

type proxyInstance struct {
	mu       sync.Mutex
	home     string
	instance *instance
}

func (inst *proxyInstance) URI() string {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.instance == nil {
		inst.instance = instanceByName(inst.home)

		if inst.instance == nil {
			return ""
		}
	}

	return inst.instance.URI
}

func (inst *proxyInstance) Token() (*oauth2.Token, error) {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.instance == nil {
		inst.instance = instanceByName(inst.home)

		if inst.instance == nil {
			return nil, fmt.Errorf("unknown instance: %s", inst.home)
		}
	}

	return inst.instance.Token()
}
