package homeassistant

import (
	"fmt"
	"sync"

	"golang.org/x/oauth2"
)

type proxyInstance struct {
	mu        sync.Mutex
	home, uri string
	oauth2.TokenSource
}

func (inst *proxyInstance) URI() string {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.uri == "" && inst.home != "" {
		// Try to resolve home name to URI (backward compatibility)
		inst.uri = instanceUriByName(inst.home)

		if inst.uri == "" {
			return ""
		}
	}

	return inst.uri
}

func (inst *proxyInstance) Token() (*oauth2.Token, error) {
	uri := inst.URI()
	if uri == "" {
		if inst.home != "" {
			return nil, fmt.Errorf("unknown instance: %s", inst.home)
		}
		return nil, fmt.Errorf("no URI configured")
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.TokenSource == nil {
		ts, err := NewHomeAssistant(uri)
		if err != nil {
			return nil, err
		}

		inst.TokenSource = ts
	}

	return inst.TokenSource.Token()
}
