package homeassistant

import (
	"sync"

	"golang.org/x/oauth2"
)

type instance struct {
	URI string
	oauth2.TokenSource
}

var (
	mu        sync.Mutex
	instances = make(map[string]*instance)
)

func instanceByName(name string) *instance {
	mu.Lock()
	defer mu.Unlock()
	return instances[name]
}
