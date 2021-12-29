package porsche

import (
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"golang.org/x/oauth2"
)

const (
	OAuthURI = "https://login.porsche.com"
)

// https://login.porsche.com/.well-known/openid-configuration
var (
	OAuth2Config = &oauth2.Config{
		ClientID:    "4mPO3OE5Srjb1iaUGWsbqKBvvesya8oA",
		RedirectURL: "https://my.porsche.com/core/de/de_DE/",
		Endpoint: oauth2.Endpoint{
			AuthURL:  OAuthURI + "/as/authorization.oauth2",
			TokenURL: OAuthURI + "/as/token.oauth2",
		},
		Scopes: []string{"openid"},
	}

	EmobilityOAuth2Config = &oauth2.Config{
		ClientID:    "NJOxLv4QQNrpZnYQbb7mCvdiMxQWkHDq",
		RedirectURL: "https://my.porsche.com/myservices/auth/auth.html",
		Endpoint:    OAuth2Config.Endpoint,
		Scopes:      OAuth2Config.Scopes,
	}
)

// Identity is the Porsche Identity client
type Identity struct {
	tr              *tokenRefresher
	DefaultSource   oauth2.TokenSource
	EmobilitySource oauth2.TokenSource
}

// NewIdentity creates Porsche identity
func NewIdentity(log *util.Logger, user, password string) *Identity {
	v := &Identity{
		tr: newTokenRefresher(log, user, password),
	}

	return v
}

func (v *Identity) Login() error {
	_, err := v.tr.RefreshToken(nil)

	if err == nil {
		v.DefaultSource = oauth.RefreshTokenSource(v.tr.DefaultToken, v.tr)
		v.EmobilitySource = oauth.RefreshTokenSource(v.tr.EmobilityToken, &emobilityAdapter{v.tr})

		v.tr.DefaultToken.Expiry = time.Now()
	}

	return err
}
