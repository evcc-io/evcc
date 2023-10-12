package autonomic

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// https://accounts.autonomic.ai/v1/auth/oidc/.well-known/openid-configuration/
var OAuth2Config = &oauth2.Config{
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://accounts.autonomic.ai/v1/auth/oidc/authorize",
		TokenURL: "https://accounts.autonomic.ai/v1/auth/oidc/token",
	},
	Scopes: []string{"openid"},
}

type Identity struct {
	*request.Helper
	ts oauth2.TokenSource
	oauth2.TokenSource
}

// NewIdentity creates autonomic token source
func NewIdentity(log *util.Logger, ts oauth2.TokenSource) (*Identity, error) {
	v := &Identity{
		Helper: request.NewHelper(log),
		ts:     ts,
	}

	token, err := ts.Token()
	if err != nil {
		return nil, err
	}

	auto, err := v.exchange(token)
	if err != nil {
		return nil, err
	}

	v.TokenSource = oauth.RefreshTokenSource(auto, v)

	return v, nil
}

// exchange authenticates with username/password to get new aws credentials
func (v *Identity) exchange(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"subject_token":      {token.AccessToken},
		"subject_issuer":     {"fordpass"},
		"client_id":          {"fordpass-prod"},
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:jwt"},
	}

	var auto *oauth.Token
	req, err := request.New(http.MethodPost, OAuth2Config.Endpoint.TokenURL, strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		err = v.DoJSON(req, &auto)
	}

	return (*oauth2.Token)(auto), err
}

// RefreshToken implements oauth.TokenRefresher
func (v *Identity) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	token, err := v.ts.Token()
	if err != nil {
		return nil, err
	}

	return v.exchange(token)
}
