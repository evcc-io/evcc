package mercedes

import (
	"fmt"
	"sync"

	"golang.org/x/oauth2"
)

var ErrNotLoggedIn = fmt.Errorf("not logged in")

type ReuseTokenSource struct {
	mu sync.Mutex
	t  *oauth2.Token
	cb func()
}

func (ts *ReuseTokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.t.Valid() {
		return ts.t, nil
	}

	ts.cb() // invalid token callback

	return nil, ErrNotLoggedIn
}

func (ts *ReuseTokenSource) Apply(t *oauth2.Token) {
	ts.mu.Lock()
	ts.t = t
	ts.mu.Unlock()
}
