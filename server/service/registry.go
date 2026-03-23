package service

import (
	"net/http"
	"sync"
)

var (
	mu       sync.Mutex
	registry = make(map[string]http.Handler)
)

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
