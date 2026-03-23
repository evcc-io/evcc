package plugin

import "sync"

var (
	httpMu      sync.Mutex
	httpMutexes = map[string]*sync.Mutex{}
)

func muForKey(key string) *sync.Mutex {
	httpMu.Lock()
	defer httpMu.Unlock()

	if mu, ok := httpMutexes[key]; ok {
		return mu
	}

	mu := new(sync.Mutex)
	httpMutexes[key] = mu
	return mu
}
