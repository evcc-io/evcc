package providerauth

import (
	"crypto/rand"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/mux"
)

var instance *Handler

type AuthProvider struct {
	ID            string `json:"id"`
	Authenticated bool   `json:"authenticated"`
}

func init() {
	var secret [16]byte
	if _, err := io.ReadFull(rand.Reader, secret[:]); err != nil {
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

// Register registers a specific AuthProvider. Returns login path as string.
func Register(name string, handler api.AuthProvider) error {
	return instance.register(name, handler)
}
