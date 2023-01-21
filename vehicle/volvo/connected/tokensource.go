package connected

import (
	"context"
	"sync"

	"golang.org/x/oauth2"
)

type ReuseTokenSource struct {
	mu sync.Mutex
	oc *oauth2.Config
	ts oauth2.TokenSource
	cb func()
}

func (ts *ReuseTokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	t, err := ts.ts.Token()
	if err != nil || !t.Valid() {
		// invalid token callback
		ts.cb()
	}

	return t, err
}

func (ts *ReuseTokenSource) Apply(t *oauth2.Token) {
	ts.mu.Lock()
	ts.ts = ts.oc.TokenSource(context.Background(), t)
	ts.mu.Unlock()
}
