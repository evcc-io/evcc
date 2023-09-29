package oauth

import (
	"errors"
	"sync"

	"dario.cat/mergo"
	"golang.org/x/oauth2"
)

type TokenRefresher interface {
	RefreshToken(token *oauth2.Token) (*oauth2.Token, error)
}

type TokenSource struct {
	mu        sync.Mutex
	token     *oauth2.Token
	refresher TokenRefresher
}

func RefreshTokenSource(token *oauth2.Token, refresher TokenRefresher) oauth2.TokenSource {
	ts := &TokenSource{
		token:     token,
		refresher: refresher,
	}

	return ts
}

func (ts *TokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.token.Valid() {
		return ts.token, nil
	}

	token, err := ts.refresher.RefreshToken(ts.token)
	if err != nil {
		return ts.token, err
	}

	if token.AccessToken == "" {
		err = errors.New("token refresh failed to obtain access token")
	} else {
		err = ts.mergeToken(token)
	}

	return ts.token, err
}

// mergeToken updates a token while preventing wiping the refresh token
func (ts *TokenSource) mergeToken(t *oauth2.Token) error {
	return mergo.Merge(ts.token, t, mergo.WithOverride)
}
