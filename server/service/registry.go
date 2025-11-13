package service

import (
	"fmt"
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

func Handler(base string) http.Handler {
	mux := http.NewServeMux()

	for name, h := range registry {
		// e.g. "/homes/foo"
		prefix := fmt.Sprintf("%s/%s", base, name)

		// strip "/homes/foo" then hand off to h
		mux.Handle(prefix+"/", http.StripPrefix(prefix, h))
	}

	return mux
}
