package providerauth

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/mux"
)

var instance *Handler

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

type AuthProvider struct {
	ID            string `json:"id"`
	Authenticated bool   `json:"authenticated"`
}

func init() {
	var secret [16]byte
	_, err := io.ReadFull(rand.Reader, secret[:])

	if err != nil {
		panic(err)
	}

	instance = &Handler{
		mu:        sync.Mutex{},
		secret:    secret[:],
		providers: make(map[string]api.AuthProvider),
		states:    make(map[string]string),
		log:       util.NewLogger("providerauth"),
	}
}

// Setup connects the redirect handler to the router and registers the callback channel
func Setup(router *mux.Router, paramC chan<- util.Param) {
	// callback?code=...&state=...
	router.Methods(http.MethodGet).Path("/callback").HandlerFunc(instance.handleCallback)
	// login?id=...
	router.Methods(http.MethodGet).Path("/login").HandlerFunc(instance.handleLogin)
	// logout?id=...
	router.Methods(http.MethodGet).Path("/logout").HandlerFunc(instance.handleLogout)

	ticker := time.NewTicker(10 * time.Second)

	go func() {
		for range ticker.C {
			instance.Publish(paramC)
		}
	}()
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

	a.log.DEBUG.Printf("publishing %d auth providers", len(apMap))

	// publish the updated auth providers
	paramC <- util.Param{Key: keys.AuthProviders, Val: apMap}
}

// Register registers a specific AuthProvider. Returns login path as string.
func Register(handler api.AuthProvider, name string) error {
	return instance.register(handler, name)
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

	// Build authorization URL
	loginUri := provider.Login(encryptedState)

	responseVal := struct {
		LoginUri string `json:"loginUri"`
	}{
		LoginUri: loginUri,
	}
	if err := json.NewEncoder(w).Encode(responseVal); err != nil {
		a.log.ERROR.Printf("failed to encode login URI response: %v", err)
	}
	w.WriteHeader(http.StatusFound)
}

func (a *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Find corresponding provider
	q := r.URL.Query()
	id := q.Get("id")
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
