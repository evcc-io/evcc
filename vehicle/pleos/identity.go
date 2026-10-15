package pleos

import (
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// tokenExpiry is assumed when the token response carries no expiry; the Pleos
// docs state access tokens are valid for one hour
const tokenExpiry = 3600

type identity struct {
	*request.Helper
	brand, clientID, clientSecret string
}

// NewIdentity creates a Pleos personal-token source for private users
// https://document.pleos.ai/en/api-reference/vehicle-data-api/getting-started/for-private-users/authorization
func NewIdentity(log *util.Logger, brand, clientID, clientSecret string) oauth2.TokenSource {
	v := &identity{
		Helper:       request.NewHelper(log),
		brand:        brand,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
	return oauth2.ReuseTokenSource(nil, oauth.Redacted(log, v))
}

// Token implements oauth2.TokenSource
func (v *identity) Token() (*oauth2.Token, error) {
	data := struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}{
		ClientID:     v.clientID,
		ClientSecret: v.clientSecret,
	}

	req, _ := request.New(http.MethodPost, BaseURL+"/auth/personal-token", request.MarshalJSON(data), map[string]string{
		"Content-Type": request.JSONContent,
		"Accept":       request.JSONContent,
		"Brand":        v.brand,
	})

	var res struct {
		Data struct {
			AccessToken string `json:"accessToken"`
			TokenType   string `json:"tokenType"`
			ExpiresIn   int64  `json:"expiresIn"`
		} `json:"data"`
	}
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	if res.Data.AccessToken == "" {
		return nil, errors.New("missing access token")
	}

	if res.Data.ExpiresIn == 0 {
		res.Data.ExpiresIn = tokenExpiry
	}

	token := &oauth2.Token{
		AccessToken: res.Data.AccessToken,
		TokenType:   res.Data.TokenType,
		ExpiresIn:   res.Data.ExpiresIn,
	}

	return util.TokenWithExpiry(token), nil
}
