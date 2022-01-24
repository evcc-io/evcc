package cmd

import (
	"sync"

	"github.com/evcc-io/evcc/util"
)

type AuthCollection struct {
	mu       sync.Mutex
	paramC   chan<- util.Param
	vehicles map[string]*AuthProvider
}

func (ac *AuthCollection) Register(title, baseURI string) *AuthProvider {
	ap := &AuthProvider{
		ac:  ac,
		Uri: baseURI,
	}

	ac.mu.Lock()
	ac.vehicles[title] = ap
	ac.mu.Unlock()

	return ap
}

// publish routes and status
func (ac *AuthCollection) Publish() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	val := struct {
		Vehicles map[string]*AuthProvider `json:"vehicles"`
	}{
		Vehicles: ac.vehicles,
	}

	ac.paramC <- util.Param{Key: "auth", Val: val}
}

type AuthProvider struct {
	ac            *AuthCollection
	Uri           string `json:"uri"`
	Authenticated bool   `json:"authenticated"`
}

func (ap *AuthProvider) Handler() chan<- bool {
	c := make(chan bool)

	go func() {
		for auth := range c {
			ap.ac.mu.Lock()
			ap.Authenticated = auth
			ap.ac.mu.Unlock()
			ap.ac.Publish()
		}
	}()

	return c
}
