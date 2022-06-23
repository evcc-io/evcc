package ford

import (
	"context"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	AuthURI       = "https://sso.ci.ford.com"
	TokenURI      = "https://api.mps.ford.com"
	ClientID      = "9fb503e0-715b-47e8-adfd-ad4b7770f73b"
	ApplicationID = "1E8C7794-FF5F-49BC-9596-A1E0C86C5B19"
)

var OAuth2Config = &oauth2.Config{
	ClientID: ClientID,
	Endpoint: oauth2.Endpoint{
		TokenURL: fmt.Sprintf("%s/oidc/endpoint/default/token", AuthURI),
	},
}

type Identity struct {
	*request.Helper
	user, password string
	oauth2.TokenSource
}

// NewIdentity creates Fiat identity
func NewIdentity(log *util.Logger, user, password string) *Identity {
	return &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}
}

// Login authenticates with username/password to get new aws credentials
func (v *Identity) Login() error {
	token, err := v.login()
	if err == nil {
		v.TokenSource = oauth.RefreshTokenSource((*oauth2.Token)(&token), v)
	}
	return err
}

// login authenticates with username/password to get new token
func (v *Identity) login() (oauth.Token, error) {
	ctx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), oauth2.HTTPClient, v.Client),
		request.Timeout,
	)
	defer cancel()

	tok, err := OAuth2Config.PasswordCredentialsToken(ctx, v.user, v.password)

	// exchange code for api token
	var token oauth.Token
	if err == nil {
		data := map[string]string{
			"ciToken": tok.AccessToken,
		}

		uri := fmt.Sprintf("%s/api/token/v2/cat-with-ci-access-token", TokenURI)

		var req *http.Request
		req, err = request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
			"Content-type":   request.JSONContent,
			"Application-Id": ApplicationID,
		})

		if err == nil {
			err = v.DoJSON(req, &token)
		}
	}

	return token, err
}

// RefreshToken implements oauth.TokenRefresher
func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := map[string]string{
		"refresh_token": token.RefreshToken,
	}

	uri := fmt.Sprintf("%s/api/token/v2/cat-with-refresh-token", TokenURI)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
		"Content-type":   request.JSONContent,
		"Application-Id": ApplicationID,
	})

	var res oauth.Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if err != nil {
		res, err = v.login()
	}

	return (*oauth2.Token)(&res), err
}
