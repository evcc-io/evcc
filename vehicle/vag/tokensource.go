package vag

import (
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api/store"
	"github.com/imdario/mergo"
	"golang.org/x/oauth2"
)

// TokenSource is a VAG token source compatible with oauth2.TokenSource
type TokenSource interface {
	// Token returns an OAuth2 compatible token (id_token omitted)
	Token() (*oauth2.Token, error)
	// TokenEx returns the extended VAG token (id_token included)
	TokenEx() (*Token, error)
}

// TokenExchanger exchanges a VW identity response into a (refreshing) VAG token source
type TokenExchanger interface {
	Exchange(q url.Values) (*Token, error)
	TokenSource(token *Token) TokenSource
}

// TokenRefreshFunc refreshes a token
type TokenRefreshFunc func(*Token) (*Token, error)

var _ TokenSource = (*tokenSource)(nil)

type tokenSource struct {
	mu    sync.Mutex
	token *Token
	new   TokenRefreshFunc
	store store.Store // TODO use or remove
}

func RefreshTokenSource(token *Token, refresher TokenRefreshFunc) *tokenSource {
	return &tokenSource{token: token, new: refresher}
}

// WithStore sets the storage option
func (v *tokenSource) WithStore(store store.Store) *tokenSource {
	v.store = store
	return v
}

// Token returns an oauth2 token or an error
func (ts *tokenSource) Token() (*oauth2.Token, error) {
	token, err := ts.TokenEx()
	if err != nil {
		return nil, err
	}

	return &token.Token, err
}

func (ts *tokenSource) TokenEx() (*Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.token == nil {
		return nil, errors.New("token not initialized")
	}

	var err error
	if time.Until(ts.token.Expiry) < time.Minute {
		if ts.new == nil {
			// this token source does not refresh
			// TODO used with AAZS, find out why
			return nil, errors.New("token source is not refreshable")
		}

		var token *Token
		if token, err = ts.new(ts.token); err == nil {
			err = ts.mergeToken(token)
		}
		if err == nil && ts.store != nil {
			err = ts.store.Save(token)
		}
	}

	return ts.token, err
}

// mergeToken updates a token while preventing wiping the refresh token
func (ts *tokenSource) mergeToken(t *Token) error {
	return mergo.Merge(ts.token, t, mergo.WithOverride)
}

type metaTokenSource struct {
	mu    sync.Mutex
	ts    TokenSource
	newT  func() (*Token, error)
	newTS func(*Token) TokenSource
}

// MetaTokenSource creates a token source that is created using the
// `newTS` function or recreated once it fails to return tokens.
// The recreation uses a new bootstrap token provided by the `newT` function.
func MetaTokenSource(newT func() (*Token, error), newTS func(*Token) TokenSource) *metaTokenSource {
	return &metaTokenSource{
		newT:  newT,
		newTS: newTS,
	}
}

// Token returns an oauth2 token or an error
func (ts *metaTokenSource) Token() (*oauth2.Token, error) {
	token, err := ts.TokenEx()
	if err != nil {
		return nil, err
	}

	return &token.Token, err
}

// Token returns a vag token or an error
func (ts *metaTokenSource) TokenEx() (*Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// use token source
	if ts.ts != nil {
		token, err := ts.ts.TokenEx()
		if err == nil {
			return token, nil
		}
	}

	// create new start token
	token, err := ts.newT()
	if err != nil {
		return nil, err
	}

	// create token source
	ts.ts = ts.newTS(token)

	// use token source
	token, err = ts.ts.TokenEx()
	if err != nil {
		// token source doesn't work anymore, reset it
		ts.ts = nil
	}

	return token, err
}
