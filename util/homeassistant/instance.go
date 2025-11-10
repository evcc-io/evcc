package homeassistant

import (
	"errors"
	"sync"

	"github.com/evcc-io/evcc/plugin/auth"
	"golang.org/x/oauth2"
)

type proxyInstance struct {
	mu       sync.Mutex
	name     string
	instance *auth.HomeAssistantInstance
}

func (inst *proxyInstance) URI() string {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.instance == nil {
		inst.instance = auth.HomeAssistantInstanceNyName(inst.name)

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
		inst.instance = auth.HomeAssistantInstanceNyName(inst.name)

		if inst.instance == nil {
			return nil, errors.New("not logged in")
		}
	}

	return inst.instance.Token()
}
