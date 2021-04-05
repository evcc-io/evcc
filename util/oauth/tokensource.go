package oauth

import (
	"errors"
	"time"

	"github.com/imdario/mergo"
	"golang.org/x/oauth2"
)

type TokenRefresher interface {
	Refresh(token *oauth2.Token) (*oauth2.Token, error)
}

type TokenSource struct {
	token     *oauth2.Token
	refresher TokenRefresher
}

func RefreshTokenSource(token *oauth2.Token, refresher TokenRefresher) oauth2.TokenSource {
	return &TokenSource{token, refresher}
}

func (ts *TokenSource) Token() (*oauth2.Token, error) {
	var err error
	if time.Until(ts.token.Expiry) < time.Minute {
		var token *oauth2.Token
		if token, err = ts.refresher.Refresh(ts.token); err == nil {
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
