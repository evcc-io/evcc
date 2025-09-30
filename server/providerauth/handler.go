package providerauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/util"
)

// Handler manages a dynamic map of routes for handling the redirect during
// OAuth authentication. When a route is registered a token OAuth state is returned.
// On GET request the generic handler identifies route and target handler
// by request state obtained from the request and delegates to the registered handler.
type Handler struct {
	mu        sync.Mutex
	secret    []byte
	providers map[string]api.AuthProvider
	states    map[string]string
	log       *util.Logger
}

func (a *Handler) Publish(paramC chan<- util.Param) {
	a.mu.Lock()
	defer a.mu.Unlock()

	apMap := make(map[string]*AuthProvider)

	for id, provider := range a.providers {
		ap := &AuthProvider{
			ID:            url.QueryEscape(id),
			Authenticated: provider.Authenticated(),
		}
		apMap[provider.DisplayName()] = ap
	}

	a.log.TRACE.Printf("publishing %d auth providers", len(apMap))

	// publish the updated auth providers
	paramC <- util.Param{Key: keys.AuthProviders, Val: apMap}
}

func (a *Handler) register(handler api.AuthProvider, name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.providers[name] != nil {
		a.log.ERROR.Printf("provider with name %s already registered", name)
		return errors.New("provider already registered")
	}

	a.log.INFO.Printf("registering oauth provider: %s", name)
	a.providers[name] = handler
	return nil
}

func (a *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Find corresponding provider
	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "missing id")
		return
	}

	a.log.DEBUG.Printf("login request for provider: %s", id)

	a.mu.Lock()
	defer a.mu.Unlock()

	provider, ok := a.providers[id]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid id")
		return
	}

	// Generate a new state and store the provider
	state := util.NewState()
	encryptedState := state.Encrypt(a.secret)
	a.states[encryptedState] = id

	// Schedule cleanup for stale state entries after state becomes invalid
	go func(state string) {
		time.Sleep(util.StateValidity)
		a.mu.Lock()
		defer a.mu.Unlock()
		delete(a.states, state)
	}(encryptedState)

	// return authorization URL
	res := struct {
		LoginUri string `json:"loginUri"`
	}{
		LoginUri: provider.Login(encryptedState),
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		a.log.ERROR.Printf("failed to encode login URI response: %v", err)
	}

	w.WriteHeader(http.StatusFound)
}

func (a *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "missing id")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	provider, ok := a.providers[id]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid id")
		return
	}

	// Handle logout
	if err := provider.Logout(); err != nil {
		a.log.ERROR.Printf("logout for provider %s failed: %v", id, err)
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *Handler) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if q.Has("error") {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: %s: %s\n", q.Get("error"), q.Get("error_description"))
		return
	}

	encryptedState := q.Get("state")
	state, err := util.DecryptState(encryptedState, a.secret)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "failed to decrypt state")
		return
	}

	if err := state.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid state")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Find the corresponding provider
	id, ok := a.states[encryptedState]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "no provider found for state")
		return
	}

	provider, ok := a.providers[id]
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal provider state unexpected")
		return
	}

	// Remove the state from the map
	delete(a.states, encryptedState)

	// Handle the callback
	if err := provider.HandleCallback(r.URL.Query()); err != nil {
		a.log.ERROR.Printf("callback handling for provider %s failed: %v", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "callback handling failed")
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
