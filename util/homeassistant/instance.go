package homeassistant

import (
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

func init() {
	auth.Register("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a Home Assistant token source from configuration
func NewHomeAssistantFromConfig(other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		URI      string
		Home     string // TODO remove deprecated
		Insecure bool
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	uri := cc.URI

	if uri == "" && cc.Home != "" {
		uri = instanceUriByName(cc.Home)
		if uri == "" {
			return nil, fmt.Errorf("unknown instance: %s", cc.Home)
		}
	}

	if ts, ok := supervisorTokenSource(uri); ok {
		return ts, nil
	}

	return NewOAuth(uri, cc.Insecure)
}

type proxyInstance struct {
	mu        sync.Mutex
	home, uri string
	insecure  bool
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
		if ts, ok := supervisorTokenSource(uri); ok {
			inst.TokenSource = ts
		} else {
			ts, err := NewOAuth(uri, inst.insecure)
			if err != nil {
				return nil, err
			}
			inst.TokenSource = ts
		}
	}

	return inst.TokenSource.Token()
}
