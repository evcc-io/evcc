package oauth

import (
	"errors"
	"sync"

	"golang.org/x/oauth2"
)

type refreshTokenSource struct {
	mu        sync.Mutex
	token     *oauth2.Token
	refresher func(token *oauth2.Token) (*oauth2.Token, error)
}

func RefreshTokenSource(token *oauth2.Token, refresher func(token *oauth2.Token) (*oauth2.Token, error)) oauth2.TokenSource {
	if token == nil {
		// allocate an (expired) token or mergeToken will fail
		token = new(oauth2.Token)
	}

	ts := &refreshTokenSource{
		token:     token,
		refresher: refresher,
	}

	return ts
}

func (ts *refreshTokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.token.Valid() {
		return ts.token, nil
	}

	token, err := ts.refresher(ts.token)
	if err != nil {
		return ts.token, err
	}

	if token.AccessToken == "" {
		return nil, errors.New("token refresh failed to obtain access token")
	}

	if token.RefreshToken == "" {
		token.RefreshToken = ts.token.RefreshToken
	}

	ts.token = token

	return ts.token, nil
}
