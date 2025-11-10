package homeassistant

import (
	"errors"
	"sync"

	"github.com/evcc-io/evcc/plugin/auth"
	"golang.org/x/oauth2"
)

type instance struct {
	mu       sync.Mutex
	name     string
	instance *auth.HomeAssistantInstance
}

type instanceAccessor interface {
	URI() string
	oauth2.TokenSource
}

func (inst *instance) URI() string {
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

func (inst *instance) Token() (*oauth2.Token, error) {
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
