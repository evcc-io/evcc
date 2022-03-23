package vag

import (
	"sync"
	"time"

	"github.com/imdario/mergo"
	"golang.org/x/oauth2"
)

type TokenRefresher func(*Token) (*Token, error)

type TokenSource interface {
	// Token returns an OAuth2 compatible token (id_token omitted)
	Token() (*oauth2.Token, error)
	// TokenEx returns the extended VAG token (id_token included)
	TokenEx() (*Token, error)
}

var _ TokenSource = (*tokenSource)(nil)

type tokenSource struct {
	mu    sync.Mutex
	token *Token
	new   TokenRefresher
}

func RefreshTokenSource(token *Token, refresher TokenRefresher) *tokenSource {
	return &tokenSource{token: token, new: refresher}
}

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

	var err error
	if time.Until(ts.token.Expiry) < time.Minute {
		var token *Token
		if token, err = ts.new(ts.token); err == nil {
			err = ts.mergeToken(token)
		}
	}

	return ts.token, err
}

// mergeToken updates a token while preventing wiping the refresh token
func (ts *tokenSource) mergeToken(t *Token) error {
	return mergo.Merge(ts.token, t, mergo.WithOverride)
}
