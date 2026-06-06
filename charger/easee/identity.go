package easee

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/cache"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// Token is the Easee Token
type Token struct {
	AccessToken  string  `json:"accessToken"`
	ExpiresIn    float32 `json:"expiresIn"`
	TokenType    string  `json:"tokenType"`
	RefreshToken string  `json:"refreshToken"`
}

func (t *Token) AsOAuth2Token() *oauth2.Token {
	if t == nil {
		return nil
	}

	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		TokenType:    t.TokenType,
		RefreshToken: t.RefreshToken,
		Expiry:       time.Now().Add(time.Second * time.Duration(t.ExpiresIn)),
	}
}

// tokenSource is an oauth2.TokenSource and holds Easee account credentials and performs authentication.
type tokenSource struct {
	*request.Helper
	user, password string
}

// NewIdentity creates an Identity for the given credentials.
func NewIdentity(log *util.Logger, user, password string) *tokenSource {
	return &tokenSource{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}
}

// tokenSourceCache stores per-account token sources
var tokenSourceCache = cache.New[oauth2.TokenSource]()

// TokenSource returns a shared oauth2.TokenSource for the given user.
func TokenSource(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	return tokenSourceCache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
		id := NewIdentity(log, user, password)
		token, err := id.Authenticate()
		if err != nil {
			return nil, err
		}
		return oauth.RefreshTokenSource(token, id.RefreshToken), nil
	})
}

// TokenSourceUncached returns a short-lived oauth2.TokenSource for the given credentials.
func TokenSourceUncached(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	id := NewIdentity(log, user, password)
	token, err := id.Authenticate()
	if err != nil {
		return nil, err
	}
	return oauth.RefreshTokenSource(token, id.RefreshToken), nil
}

// Authenticate performs the initial username/password login and returns an oauth2.Token.
func (c *tokenSource) Authenticate() (*oauth2.Token, error) {
	data := struct {
		Username string `json:"userName"`
		Password string `json:"password"`
	}{
		Username: c.user,
		Password: c.password,
	}

	uri := fmt.Sprintf("%s/%s", API, "accounts/login")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return nil, err
	}

	var token Token
	if err := c.DoJSON(req, &token); err != nil {
		return nil, err
	}

	return token.AsOAuth2Token(), nil
}

// RefreshToken refreshes an existing oauth2 token using the Easee refresh endpoint.
// Falls back to a full re-login when the refresh endpoint rejects the token.
func (c *tokenSource) RefreshToken(oauthToken *oauth2.Token) (*oauth2.Token, error) {
	data := struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}{
		AccessToken:  oauthToken.AccessToken,
		RefreshToken: oauthToken.RefreshToken,
	}

	uri := fmt.Sprintf("%s/%s", API, "accounts/refresh_token")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return nil, err
	}

	var token *Token
	if err := c.DoJSON(req, &token); err != nil {
		if c.password == "" {
			return nil, api.ErrCredentialsRequired
		}
		// re-login on refresh failure when the password is still available in memory.
		return c.Authenticate()
	}

	return token.AsOAuth2Token(), nil
}

func easeeAccountSubject(user string) string {
	h := sha256.Sum256([]byte(user))
	return "easee-" + hex.EncodeToString(h[:])[:8]
}

type persistedEaseeAuth struct {
	User  string        `json:"user"`
	Token *oauth2.Token `json:"token"`
}

func loadPersistedEaseeAuth(subject string) *persistedEaseeAuth {
	if !settings.Exists(subject) {
		return nil
	}

	var stored persistedEaseeAuth
	if err := settings.Json(subject, &stored); err != nil {
		return nil
	}
	if stored.Token == nil || stored.Token.RefreshToken == "" {
		return nil
	}

	return &stored
}

// persistEaseeToken saves the token to the DB so it can be reused across restarts without a fresh login.
func persistEaseeToken(log *util.Logger, subject, user string, token *oauth2.Token) {
	if token == nil {
		return
	}

	payload := persistedEaseeAuth{
		User:  user,
		Token: token,
	}

	if err := settings.SetJson(subject, payload); err != nil {
		log.WARN.Printf("failed to persist Easee token: %v", err)
	}
}

func HasPersistedAuth(user string) bool {
	if user == "" {
		return false
	}

	return loadPersistedEaseeAuth(easeeAccountSubject(user)) != nil
}

func PersistentTokenSource(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	if user == "" {
		return nil, api.ErrMissingCredentials
	}

	subject := easeeAccountSubject(user)
	stored := loadPersistedEaseeAuth(subject)

	if stored == nil && password == "" {
		return nil, api.ErrCredentialsRequired
	}

	currentUser := user
	currentPassword := password
	if stored != nil {
		if currentUser == "" {
			currentUser = stored.User
		}
	}

	initial := (*oauth2.Token)(nil)
	if stored != nil {
		initial = stored.Token
	}

	// Pre-seed the inner cache with the DB token so TokenSource callers skip login
	if initial != nil {
		id := NewIdentity(log, currentUser, currentPassword)
		_, _ = tokenSourceCache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
			return oauth.RefreshTokenSource(initial, id.RefreshToken), nil
		})
	}

	refreshWithPersist := func(_ *oauth2.Token) (*oauth2.Token, error) {
		base, err := TokenSource(log, currentUser, currentPassword)
		if err != nil {
			return nil, err
		}
		newToken, err := base.Token()
		if err != nil {
			return nil, err
		}
		persistEaseeToken(log, subject, currentUser, newToken)
		return newToken, nil
	}

	return oauth.RefreshTokenSource(initial, refreshWithPersist), nil
}
