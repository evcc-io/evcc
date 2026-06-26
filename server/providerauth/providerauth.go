package providerauth

import (
	"crypto/rand"
	"io"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/mux"
)

var instance *Handler

type AuthProvider struct {
	ID            string `json:"id"`
	Authenticated bool   `json:"authenticated"`
	Interactive   bool   `json:"interactive,omitempty"` // credential form instead of redirect/device flow
}

// Challenge returns the initial interactive-login challenge for the provider.
func Challenge(id string) (*api.AuthChallenge, error) {
	return instance.challenge(id)
}

// Submit processes interactive-login field values for the provider and returns
// the next challenge, or done=true when authentication succeeded.
func Submit(id string, values map[string]string) (challenge *api.AuthChallenge, done bool, err error) {
	return instance.submit(id, values)
}

func init() {
	var secret [16]byte
	if _, err := io.ReadFull(rand.Reader, secret[:]); err != nil {
		panic(err)
	}

	instance = &Handler{
		log:       util.NewLogger("providerauth"),
		secret:    secret[:],
		providers: make(map[string]api.AuthProvider),
		states:    make(map[string]stateEntry),
		updateC:   make(chan string, 1),
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

	go instance.run(paramC)
}

// Register registers a specific AuthProvider by name
// The returned online channel is used to asynchronously update authorization status
func Register(name string, handler api.AuthProvider) (chan<- bool, error) {
	updateC, err := instance.register(name, handler)
	if err != nil {
		return nil, err
	}

	// buffered + non-blocking send (see OAuth.setOnline): the value is only a
	// signal and the handler re-reads live state, so coalescing is lossless.
	onlineC := make(chan bool, 1)

	go func() {
		for range onlineC {
			updateC <- name
		}
	}()

	return onlineC, nil
}
