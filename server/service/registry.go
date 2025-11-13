package service

import (
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
	handler := &handler{
		base: base,
	}

	mux := http.NewServeMux()
	mux.Handle(base+"/{service}/", handler)

	return mux
}

type handler struct {
	base string
}

// func (h *handler) HandleService(w http.ResponseWriter, req *http.Request) {
func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	service := req.PathValue("service")

	handler, ok := registry[service]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req.URL.Path = strings.TrimPrefix(req.URL.Path, h.base+"/"+service)

	handler.ServeHTTP(w, req)
}
