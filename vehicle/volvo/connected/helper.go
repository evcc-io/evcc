package connected

import (
	"sync"

	"golang.org/x/oauth2"
)

var (
	mu         sync.Mutex
	identities = make(map[string]oauth2.TokenSource)
)

func getInstance(subject string) oauth2.TokenSource {
	return identities[subject]
}

func addInstance(subject string, identity oauth2.TokenSource) {
	identities[subject] = identity
}
