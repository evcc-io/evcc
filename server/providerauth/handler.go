package providerauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/util"
)

type errorResponse struct {
	Error string `json:"error"`
}

type loginResponse struct {
	LoginUri string     `json:"loginUri"`
	Code     string     `json:"code,omitempty"`
	Expiry   *time.Time `json:"expiry,omitempty"`
}

// jsonWrite writes a JSON response
func jsonWrite(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// jsonError writes an error response
func jsonError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	jsonWrite(w, errorResponse{Error: message})
}

// Handler manages a dynamic map of routes for handling the redirect during
// OAuth authentication. When a route is registered a token OAuth state is returned.
// On GET request the generic handler identifies route and target handler
// by request state obtained from the request and delegates to the registered handler.
type Handler struct {
	mu        sync.Mutex
	log       *util.Logger
	secret    []byte
	providers map[string]api.AuthProvider
	states    map[string]stateEntry
	updateC   chan string
}

type stateEntry struct {
	id       string
	returnTo string // config modal query to restore on callback
}

// TODO get status from update channel
func (a *Handler) run(paramC chan<- util.Param) {
	for range a.updateC {
		a.mu.Lock()

		res := make(map[string]*AuthProvider)
		for id, provider := range a.providers {
			res[provider.DisplayName()] = &AuthProvider{
				ID:            id,
				Authenticated: provider.Authenticated(),
			}
		}

		a.mu.Unlock()

		// publish the updated auth providers
		paramC <- util.Param{Key: keys.AuthProviders, Val: res}
	}
}

func (a *Handler) register(name string, handler api.AuthProvider) (chan<- string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.providers[name] != nil {
		return nil, fmt.Errorf("provider already registered: %s", name)
	}

	a.providers[name] = handler

	return a.updateC, nil
}

func (a *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	a.log.DEBUG.Printf("login request for: %s", id)

	a.mu.Lock()
	defer a.mu.Unlock()

	provider, ok := a.providers[id]
	if !ok {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Generate a new state and store the provider
	state := NewState()
	encryptedState := state.Encrypt(a.secret)
	a.states[encryptedState] = stateEntry{id: id, returnTo: r.URL.Query().Get("return")}

	// Schedule cleanup for stale state entries after state becomes invalid
	time.AfterFunc(stateValidity, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		delete(a.states, encryptedState)
	})

	uri, da, err := provider.Login(encryptedState)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	res := loginResponse{
		LoginUri: uri,
	}

	if da != nil {
		res.Expiry = &da.Expiry
		if da.VerificationURIComplete != "" {
			res.LoginUri = da.VerificationURIComplete
		} else {
			res.LoginUri = da.VerificationURI
			res.Code = da.UserCode
		}
	}

	jsonWrite(w, res)
}

func (a *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	a.log.DEBUG.Printf("logout request for: %s", id)

	a.mu.Lock()
	defer a.mu.Unlock()

	provider, ok := a.providers[id]
	if !ok {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Handle logout
	if err := provider.Logout(); err != nil {
		a.log.ERROR.Printf("logout for provider %s failed: %v", id, err)
		jsonError(w, http.StatusInternalServerError, "logout failed")
		return
	}

	jsonWrite(w, "OK")
}

func (a *Handler) redirectToError(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, "/#/config?callbackError="+url.QueryEscape(message), http.StatusFound)
}

func (a *Handler) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if q.Has("error") {
		errorMsg := q.Get("error") + ": " + q.Get("error_description")
		a.redirectToError(w, r, errorMsg)
		return
	}

	encryptedState := q.Get("state")
	state, err := DecryptState(encryptedState, a.secret)
	if err != nil || !state.Valid() {
		a.redirectToError(w, r, "invalid state")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Find the corresponding provider
	entry, ok := a.states[encryptedState]
	if !ok {
		a.redirectToError(w, r, "no provider found for state")
		return
	}
	id := entry.id

	provider, ok := a.providers[id]
	if !ok {
		a.redirectToError(w, r, "internal provider state unexpected")
		return
	}

	// Remove the state from the map
	delete(a.states, encryptedState)

	// Handle the callback
	if err := provider.HandleCallback(r.URL.Query()); err != nil {
		a.log.ERROR.Printf("callback for provider %s failed: %v", id, err)
		a.redirectToError(w, r, err.Error())
		return
	}

	// restore the config modal stack alongside the completion marker
	query := "callbackCompleted=" + url.QueryEscape(id)
	if entry.returnTo != "" {
		query = entry.returnTo + "&" + query
	}

	http.Redirect(w, r, "/#/config?"+query, http.StatusFound)
}
