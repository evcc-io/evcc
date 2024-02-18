package tesla

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// https://auth.tesla.com/oauth2/v3/.well-known/openid-configuration

// OAuth2Config is the OAuth2 configuration for authenticating with the Tesla API.
var OAuth2Config = &oauth2.Config{
	RedirectURL: "https://auth.tesla.com/void/callback",
	Endpoint: oauth2.Endpoint{
		AuthURL:   "https://auth.tesla.com/en_us/oauth2/v3/authorize",
		TokenURL:  "https://auth.tesla.com/oauth2/v3/token",
		AuthStyle: oauth2.AuthStyleInParams,
	},
	Scopes: []string{"openid", "email", "offline_access"},
}

var TESLA_CLIENT_ID, TESLA_CLIENT_SECRET string

func init() {
	if TESLA_CLIENT_ID != "" {
		OAuth2Config.ClientID = TESLA_CLIENT_ID
	}
	if TESLA_CLIENT_SECRET != "" {
		OAuth2Config.ClientSecret = TESLA_CLIENT_SECRET
	}
}

type Identity struct {
	oauth2.TokenSource
	mu      sync.Mutex
	log     *util.Logger
	subject string
}

func NewIdentity(log *util.Logger, token *oauth2.Token) (oauth2.TokenSource, error) {
	// serialise instance handling
	mu.Lock()
	defer mu.Unlock()

	// determine tesla identity
	var claims jwt.RegisteredClaims
	if _, _, err := jwt.NewParser().ParseUnverified(token.AccessToken, &claims); err != nil {
		return nil, err
	}

	// reuse identity instance
	if instance := getInstance(claims.Subject); instance != nil {
		return instance, nil
	}

	if !token.Valid() {
		token.Expiry = claims.ExpiresAt.Time
	}

	v := &Identity{
		log:     log,
		subject: claims.Subject,
	}

	// database token
	if !token.Valid() {
		var tok oauth2.Token
		if err := settings.Json(v.settingsKey(), &tok); err == nil {
			token = &tok
		}
	}

	if !token.Valid() && token.RefreshToken != "" {
		if tok, err := v.RefreshToken(token); err == nil {
			token = tok
		}
	}

	if !token.Valid() {
		return nil, errors.New("token expired")
	}

	v.TokenSource = oauth.RefreshTokenSource(token, v)

	// add instance
	addInstance(claims.Subject, v)

	return v, nil
}

func (v *Identity) settingsKey() string {
	return fmt.Sprintf("tesla-command.%s", v.subject)
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(v.log))
	ts := OAuth2Config.TokenSource(ctx, token)

	token, err := ts.Token()
	if err != nil {
		return nil, err
	}

	err = settings.SetJson(v.settingsKey(), token)

	return token, err
}
