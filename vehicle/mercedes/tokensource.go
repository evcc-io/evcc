package mercedes

import (
	"fmt"
	"sync"

	"golang.org/x/oauth2"
)

var (
	ErrNotLoggedIn = fmt.Errorf("not logged in")
	ErrExpired     = fmt.Errorf("token expired")
)

type ExpiringSource struct {
	mu sync.Mutex
	t  *oauth2.Token
}

func (ts *ExpiringSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	var err error
	if ts.t == nil {
		err = ErrNotLoggedIn
	} else if !ts.t.Valid() {
		err = ErrExpired
	}

	return ts.t, err
}

func (ts *ExpiringSource) Apply(t *oauth2.Token) {
	ts.mu.Lock()
	ts.t = t
	ts.mu.Unlock()
}
