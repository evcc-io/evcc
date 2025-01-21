package util

import "sync"

type AuthCollection struct {
	mu       sync.Mutex
	paramC   chan<- Param
	vehicles map[string]*AuthProvider
}

func NewAuthCollection(paramC chan<- Param) *AuthCollection {
	return &AuthCollection{
		paramC:   paramC,
		vehicles: make(map[string]*AuthProvider),
	}
}

func (ac *AuthCollection) Register(baseURI, title string) *AuthProvider {
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
		Vehicles map[string]*AuthProvider `json:"vehicles,omitempty"`
	}{
		Vehicles: ac.vehicles,
	}

	// TODO registered global key name
	ac.paramC <- Param{Key: "auth", Val: val}
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
