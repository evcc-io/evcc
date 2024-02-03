package vc

import "sync"

var (
	mu         sync.Mutex
	identities = make(map[string]*Identity)
)

func GetInstance(subject string) *Identity {
	mu.Lock()
	defer mu.Unlock()
	v, _ := identities[subject]
	return v
}

func AddInstance(subject string, identity *Identity) {
	mu.Lock()
	defer mu.Unlock()
	identities[subject] = identity
}
