package providerauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/util"
)

type errorResponse struct {
	Error string `json:"error"`
}

// loginResponse is one of three shapes: a redirect url, a device code, or a
// challenge answered in the ui. Authenticated is set when no user input is needed.
type loginResponse struct {
	LoginUri      string             `json:"loginUri,omitempty"`
	Code          string             `json:"code,omitempty"`
	Expiry        *time.Time         `json:"expiry,omitempty"`
	Challenge     *api.AuthChallenge `json:"challenge,omitempty"`
	Authenticated bool               `json:"authenticated,omitempty"`
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

// stateEntry remembers the page the login started from, the redirect uri may be on another origin
type stateEntry struct {
	id       string
	returnTo string // e.g. http://evcc.local:7070/#/config?vehicle=8
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
	provider, ok := a.providers[id]
	if !ok {
		a.mu.Unlock()
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// server-side login needs no redirect state. runs outside the lock: it does network i/o
	if c, ok := provider.(api.AuthChallenger); ok {
		a.mu.Unlock()
		challenge, err := c.StartChallenge()
		a.challengeResponse(w, id, challenge, err)
		return
	}
	defer a.mu.Unlock()

	// Generate a new state and store the provider
	state := NewState()
	encryptedState := state.Encrypt(a.secret)
	a.states[encryptedState] = stateEntry{id: id, returnTo: returnURL(r.URL.Query().Get("return"), r.Referer())}

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

// challengeResponse writes the outcome of a challenge step
func (a *Handler) challengeResponse(w http.ResponseWriter, id string, challenge *api.AuthChallenge, err error) {
	if err != nil {
		a.log.DEBUG.Printf("login for provider %s failed: %v", id, err)
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonWrite(w, loginResponse{Challenge: challenge, Authenticated: challenge == nil})
}

func (a *Handler) handleSubmit(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	a.log.DEBUG.Printf("submit request for: %s", id)

	var req struct {
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request")
		return
	}

	a.mu.Lock()
	provider, ok := a.providers[id].(api.AuthChallenger)
	a.mu.Unlock()

	if !ok {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	challenge, err := provider.SubmitChallenge(req.Answer)
	a.challengeResponse(w, id, challenge, err)
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

// returnURL validates the page to return to after the callback, empty if unusable.
// The page must be the one the login was requested from, the callback is reachable without session.
// The browser-set referer is used since proxies (e.g. dev server) may rewrite the host.
func returnURL(s, referer string) string {
	ref, err := url.Parse(referer)
	if err != nil {
		return ""
	}
	if u, err := url.Parse(s); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host != ref.Host || u.Scheme != ref.Scheme {
		return ""
	}
	return s
}

// redirectToConfig sends the browser back to the config page the login started from
func redirectToConfig(w http.ResponseWriter, r *http.Request, returnTo, query string) {
	if returnTo == "" {
		returnTo = "/#/config"
	}
	// the query must land in the fragment for the router to see it
	sep := "?"
	if strings.Contains(returnTo[strings.LastIndex(returnTo, "#")+1:], "?") {
		sep = "&"
	}
	http.Redirect(w, r, returnTo+sep+query, http.StatusFound)
}

func (a *Handler) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// the callback is reachable without session, only act on states issued by us
	encryptedState := q.Get("state")
	state, err := DecryptState(encryptedState, a.secret)
	if err != nil || !state.Valid() {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Find the corresponding provider
	entry, ok := a.states[encryptedState]
	if !ok {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	id := entry.id

	// Remove the state from the map
	delete(a.states, encryptedState)

	redirectToError := func(message string) {
		redirectToConfig(w, r, entry.returnTo, "callbackError="+url.QueryEscape(message))
	}

	if q.Has("error") {
		redirectToError(q.Get("error") + ": " + q.Get("error_description"))
		return
	}

	provider, ok := a.providers[id]
	if !ok {
		redirectToError("internal provider state unexpected")
		return
	}

	// Handle the callback
	if err := provider.HandleCallback(q); err != nil {
		a.log.ERROR.Printf("callback for provider %s failed: %v", id, err)
		redirectToError(err.Error())
		return
	}

	redirectToConfig(w, r, entry.returnTo, "callbackCompleted="+url.QueryEscape(id))
}
