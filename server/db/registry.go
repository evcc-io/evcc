package db

import "sync"

var (
	mu       sync.Mutex
	registry []func() error
)

func Register(fun func() error) {
	mu.Lock()
	defer mu.Unlock()
	registry = append(registry, fun)
}
