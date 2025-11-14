package network

import (
	"sync"

	"github.com/evcc-io/evcc/api/globalconfig"
)

var (
	mu       sync.Mutex
	registry []func()
)

var config globalconfig.Network

func Config() globalconfig.Network {
	return config
}

func Register(fun func()) {
	mu.Lock()
	defer mu.Unlock()

	registry = append(registry, fun)
}

func Start(conf globalconfig.Network) {
	config = conf

	for _, fun := range registry {
		go fun()
	}
}
