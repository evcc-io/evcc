package iobroker

import (
	"fmt"
	"sync"

	"golang.org/x/oauth2"
)

type proxyInstance struct {
	mu  sync.Mutex
	uri string
	oauth2.TokenSource
}

func (inst *proxyInstance) URI() string {
	return inst.uri
}

func (inst *proxyInstance) Token() (*oauth2.Token, error) {
	uri := inst.URI()
	if uri == "" {
		return nil, fmt.Errorf("no URI configured")
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.TokenSource == nil {
		ts, err := NewIobroker(uri)
		if err != nil {
			return nil, err
		}

		inst.TokenSource = ts
	}

	return inst.TokenSource.Token()
}
