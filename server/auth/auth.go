package auth

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/mux"
)

var instance *Auth

type Auth struct {
	mu     sync.Mutex
	secret []byte
	routes map[string]http.HandlerFunc
}

func generateSecret() ([]byte, error) {
	var b [16]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	return b[:], err
}

func init() {
	secret, err := generateSecret()
	if err != nil {
		panic(err)
	}

	instance = &Auth{
		secret: secret,
		routes: make(map[string]http.HandlerFunc),
	}
}

func Setup(router *mux.Router) {
	router.Methods(http.MethodGet).HandlerFunc(instance.handle)
}

func Register(handler http.HandlerFunc) string {
	return instance.register(handler)
}

func (a *Auth) register(handler http.HandlerFunc) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	state := util.NewState()
	key := state.Encrypt(a.secret)

	a.routes[key] = handler

	return key
}

func (a *Auth) handle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if error, ok := vars["error"]; ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: %s: %s\n", error, vars["error_description"])
		return
	}

	state, err := util.DecryptState(vars["state"], a.secret)
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
	handler := a.routes[vars["state"]]
	a.mu.Unlock()

	if handler == nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "no handler found")
		return
	}

	handler(w, r)
}
