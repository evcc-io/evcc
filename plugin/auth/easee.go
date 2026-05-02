package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/easee"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("easee", newEaseeFromConfig)
}

var (
	easeeInstancesMu sync.Mutex
	easeeInstances   = make(map[string]oauth2.TokenSource)
)

func newEaseeFromConfig(_ context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		User     string
		Password string
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}
	return NewEaseeTokenSource(cc.User, cc.Password)
}

// easeeSubject derives a stable settings DB key from the user email and password.
// Including the password ensures that a password change triggers reauthentication
// when no valid refresh token is available.
func easeeSubject(user, password string) string {
	h := sha256.Sum256([]byte(user + ":" + password))
	return "easee-" + hex.EncodeToString(h[:])[:8]
}

// loadEaseeToken restores a persisted token from the DB to avoid an unnecessary login.
func loadEaseeToken(subject string) *oauth2.Token {
	if !settings.Exists(subject) {
		return nil
	}

	var storedToken oauth2.Token
	if err := settings.Json(subject, &storedToken); err != nil {
		return nil
	}
	if storedToken.RefreshToken == "" {
		return nil
	}

	return &storedToken
}

// persistEaseeToken saves the token to the DB so it can be reused across restarts without a fresh login.
func persistEaseeToken(log *util.Logger, subject string, token *oauth2.Token) {
	if err := settings.SetJson(subject, token); err != nil {
		log.WARN.Printf("failed to persist Easee token: %v", err)
	}
}

// NewEaseeTokenSource returns a shared, persistent oauth2.TokenSource for the
// given Easee account. Multiple callers with the same user email share one
// token source so the Easee cloud API is not called unnecessarily.
// Tokens are persisted to the settings DB so a restart does not require
// a fresh login.
func NewEaseeTokenSource(user, password string) (oauth2.TokenSource, error) {
	easeeInstancesMu.Lock()
	defer easeeInstancesMu.Unlock()

	subject := easeeSubject(user, password)
	if ts, ok := easeeInstances[subject]; ok {
		return ts, nil
	}

	log := util.NewLogger("easee").Redact(user, password)
	id := easee.NewIdentity(log, user, password)

	initialToken := loadEaseeToken(subject)
	if initialToken == nil {
		token, err := id.Authenticate()
		if err != nil {
			return nil, err
		}
		persistEaseeToken(log, subject, token)
		initialToken = token
	}

	refreshWithPersist := func(token *oauth2.Token) (*oauth2.Token, error) {
		newToken, err := id.RefreshToken(token)
		if err != nil {
			return nil, err
		}
		persistEaseeToken(log, subject, newToken)
		return newToken, nil
	}

	ts := oauth.RefreshTokenSource(initialToken, refreshWithPersist)
	easeeInstances[subject] = ts
	return ts, nil
}
