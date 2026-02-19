package oauth

import (
	"sync"

	"golang.org/x/oauth2"
)

type bootstrapTokenSource struct {
	mu        sync.Mutex
	refresher func() (*oauth2.Token, error)
}

func BootstrapTokenSource(refresher func() (*oauth2.Token, error)) oauth2.TokenSource {
	return &bootstrapTokenSource{
		refresher: refresher,
	}
}

func (ts *bootstrapTokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	return ts.refresher()
}
