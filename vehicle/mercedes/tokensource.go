package mercedes

import (
	"context"
	"reflect"
	"sync"

	"github.com/evcc-io/evcc/api/store"
	"golang.org/x/oauth2"
)

type TokenSourceProvider interface {
	TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource
}

type ReuseTokenSource struct {
	mu    sync.Mutex
	oc    TokenSourceProvider
	ts    oauth2.TokenSource
	cb    func(bool)
	store store.Store
}

// WithStore attaches a persistent store
func (ts *ReuseTokenSource) WithStore(store store.Store) *ReuseTokenSource {
	if store != nil && !reflect.ValueOf(store).IsNil() {
		ts.store = store
	}
	return ts
}

func (ts *ReuseTokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	token, err := ts.ts.Token()

	valid := err == nil && token.Valid()
	if valid && ts.store != nil {
		err = ts.store.Save(token)
	}

	// update status
	ts.cb(valid)

	return token, err
}

func (ts *ReuseTokenSource) Update(token *oauth2.Token) {
	if ts.store != nil {
		_ = ts.store.Save(token)
	}

	ts.mu.Lock()
	ts.ts = ts.oc.TokenSource(context.Background(), token)
	ts.mu.Unlock()

	// update status
	_, _ = ts.Token()
}
