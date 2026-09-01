package oauth

import (
	"errors"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

// ErrTokenExpired is returned when neither the persisted nor the given token could be used or refreshed
var ErrTokenExpired = errors.New("token expired")

type persistentTokenSource struct {
	mu        sync.Mutex
	log       *util.Logger
	key       string
	token     *oauth2.Token
	refresher func(token *oauth2.Token) (*oauth2.Token, error)
}

// PersistentTokenSource is a RefreshTokenSource that persists the token under key.
// A usable persisted token takes precedence over the given token, which typically
// stems from the device configuration.
func PersistentTokenSource(log *util.Logger, key string, token *oauth2.Token, refresher func(token *oauth2.Token) (*oauth2.Token, error)) (oauth2.TokenSource, error) {
	ts := &persistentTokenSource{
		log:       log,
		key:       key,
		refresher: refresher,
	}

	candidates := make([]*oauth2.Token, 0, 2)

	var stored oauth2.Token
	if err := settings.Json(key, &stored); err == nil {
		candidates = append(candidates, &stored)
	}

	if token != nil {
		candidates = append(candidates, token)
	}

	for _, tok := range candidates {
		if !tok.Valid() && tok.RefreshToken != "" {
			var err error
			if tok, err = ts.refresh(tok); err != nil {
				log.DEBUG.Printf("refreshing token: %v", err)
				continue
			}
		}

		if tok.Valid() {
			ts.store(tok)
			return Redacted(log, ts), nil
		}
	}

	return nil, ErrTokenExpired
}

func (ts *persistentTokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.token.Valid() {
		return ts.token, nil
	}

	token, err := ts.refresh(ts.token)
	if err != nil {
		return ts.token, err
	}

	ts.store(token)

	return ts.token, nil
}

func (ts *persistentTokenSource) refresh(token *oauth2.Token) (*oauth2.Token, error) {
	res, err := ts.refresher(token)
	if err != nil {
		// the api rejected the token itself, so keeping it would only make the next start fail again
		if rejected(err) {
			if err := settings.Delete(ts.key); err != nil {
				ts.log.ERROR.Printf("deleting token: %v", err)
			}
		}

		return nil, err
	}

	if res.AccessToken == "" {
		return nil, errors.New("token refresh failed to obtain access token")
	}

	if res.RefreshToken == "" {
		res.RefreshToken = token.RefreshToken
	}

	return res, nil
}

// store persists the token. Settings are flushed right away since apis with single-use
// refresh tokens lock the user out if the rotated token is lost before the periodic flush.
func (ts *persistentTokenSource) store(token *oauth2.Token) {
	ts.token = token

	if err := settings.SetJson(ts.key, token); err != nil {
		ts.log.ERROR.Printf("saving token: %v", err)
		return
	}

	if err := settings.Persist(); err != nil {
		ts.log.ERROR.Printf("saving token: %v", err)
	}
}

// rejected reports whether the api refused the token itself rather than failing for a transient reason
func rejected(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "invalid_grant") || strings.Contains(msg, "invalid_token")
}
