package service

import (
	"maps"
	"net/http"
	"sync"
)

var (
	mu       sync.Mutex
	registry = make(map[string]http.Handler)
	public   = make(map[string]http.Handler)
)

// RegisterPublic exposes an unauthenticated GET route on the root router,
// also through remote access, e.g. for third parties fetching well-known files
func RegisterPublic(path string, handler http.Handler) {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := public[path]; ok {
		panic("public route " + path + " already registered")
	}

	public[path] = handler
}

// PublicRoutes returns the registered public routes by path
func PublicRoutes() map[string]http.Handler {
	mu.Lock()
	defer mu.Unlock()

	return maps.Clone(public)
}

// IsPublic returns true if the path is a registered public route
func IsPublic(path string) bool {
	mu.Lock()
	defer mu.Unlock()

	_, ok := public[path]
	return ok
}

func Register(name string, handler http.Handler) {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := registry[name]; ok {
		panic("service " + name + " already registered")
	}

	registry[name] = handler
}

func Handler() http.Handler {
	mux := http.NewServeMux()

	for name, h := range registry {
		// e.g. "/homes/foo"
		prefix := "/" + name

		// strip "/homes/foo" then hand off to h
		mux.Handle(prefix+"/", http.StripPrefix(prefix, h))
	}

	return mux
}
