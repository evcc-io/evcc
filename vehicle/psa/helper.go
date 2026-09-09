package psa

import (
	"sync"
)

var (
	mu         sync.Mutex
	identities = make(map[string]*Identity)
)

func getInstance(subject string) *Identity {
	return identities[subject]
}

func addInstance(subject string, identity *Identity) {
	identities[subject] = identity
}
