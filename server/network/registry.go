package network

import (
	"sync"

	"github.com/evcc-io/evcc/api/globalconfig"
)

var (
	mu       sync.Mutex
	registry []func()
)

var Config globalconfig.Network

func Register(fun func()) {
	mu.Lock()
	defer mu.Unlock()

	registry = append(registry, fun)
}

func Start() {
	for _, fun := range registry {
		go fun()
	}
}
