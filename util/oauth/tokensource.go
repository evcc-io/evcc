package oauth

import (
	"errors"
	"sync"
	"time"

	"github.com/imdario/mergo"
	"golang.org/x/oauth2"
)

type TokenRefresher interface {
	RefreshToken(token *oauth2.Token) (*oauth2.Token, error)
}

type TokenSource struct {
	mu        sync.Mutex
	token     *oauth2.Token
	refresher TokenRefresher
	margin    time.Duration
}

type TokenSourceOption func(*TokenSource)

func WithExpiryMargin(margin time.Duration) TokenSourceOption {
	return func(ts *TokenSource) {
		ts.margin = margin
	}
}

func RefreshTokenSource(token *oauth2.Token, refresher TokenRefresher, opt ...TokenSourceOption) oauth2.TokenSource {
	ts := &TokenSource{
		token:     token,
		refresher: refresher,
		margin:    time.Minute,
	}

	for _, o := range opt {
		o(ts)
	}

	return ts
}

func (ts *TokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	var err error
	if ts.token == nil || time.Until(ts.token.Expiry) < ts.margin {
		var token *oauth2.Token
		if token, err = ts.refresher.RefreshToken(ts.token); err == nil {
			if token.AccessToken == "" {
				err = errors.New("token refresh failed to obtain access token")
			} else {
				err = ts.mergeToken(token)
			}
		}
	}
	return ts.token, err
}

// mergeToken updates a token while preventing wiping the refresh token
func (ts *TokenSource) mergeToken(t *oauth2.Token) error {
	return mergo.Merge(ts.token, t, mergo.WithOverride)
}
