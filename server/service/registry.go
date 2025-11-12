package service

import (
	"fmt"
	"net/http"
	"strings"
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
	return &handler{
		base: base,
	}
}

type handler struct {
	base string
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Println(req.URL)

	path := strings.TrimPrefix(req.URL.Path, h.base)
	segments := strings.SplitN(path, "/", 2)

	if len(segments) > 0 {
		service := segments[0]

		if handler, ok := registry[service]; ok {
			req.URL.Path = strings.Join(segments[1:], "/")
			handler.ServeHTTP(w, req)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `["foo","bar"]`)
}
