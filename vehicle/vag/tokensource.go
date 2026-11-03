package vag

import (
	"net/url"
	"sync"
	"time"

	"dario.cat/mergo"
	"github.com/evcc-io/evcc/util"
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

// TokenRefresher refreshes a token
type TokenRefresher func(*Token) (*Token, error)

var _ TokenSource = (*tokenSource)(nil)

type tokenSource struct {
	mu     sync.Mutex
	token  *Token
	new    TokenRefresher
	redact func(...string)
}

func RefreshTokenSource(log *util.Logger, token *Token, refresher TokenRefresher) *tokenSource {
	ts := &tokenSource{token: token, new: refresher, redact: log.RotatingSlot()}
	ts.redactToken()

	return ts
}

// redactToken keeps the current tokens out of the logs, where they would
// otherwise show up in the Authorization header
func (ts *tokenSource) redactToken() {
	ts.redact(ts.token.AccessToken, ts.token.RefreshToken, ts.token.IDToken)
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

	var err error
	if time.Until(ts.token.Expiry) < time.Minute {
		var token *Token
		if token, err = ts.new(ts.token); err == nil {
			if err = ts.mergeToken(token); err == nil {
				ts.redactToken()
			}
		}
	}

	return ts.token, err
}

// mergeToken updates a token while preventing wiping the refresh token
func (ts *tokenSource) mergeToken(t *Token) error {
	return mergo.Merge(ts.token, t, mergo.WithOverride)
}
