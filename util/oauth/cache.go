package oauth

import (
	"github.com/evcc-io/evcc/api/store"
	"golang.org/x/oauth2"
)

var _ oauth2.TokenSource = (*cachedTokenSource)(nil)

type cachedTokenSource struct {
	ts    oauth2.TokenSource
	store store.Store
}

func CachedTokenSource(store store.Store, ts oauth2.TokenSource) oauth2.TokenSource {
	return &cachedTokenSource{
		ts:    ts,
		store: store,
	}
}

func (ts *cachedTokenSource) Token() (*oauth2.Token, error) {
	token, err := ts.ts.Token()
	if err == nil {
		err = ts.store.Save(token)
	}
	return token, err
}
