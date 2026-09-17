package polestar

import (
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// scopes requested for the capabilities evcc exposes
var scopes = []string{
	"pdp-telemetry/battery",
	"pdp-telemetry/odometer",
	"pdp-charging/targetSoc",
}

type identity struct {
	*request.Helper
	clientID, clientSecret string
	uri                    string
}

// NewIdentity creates a Polestar Data Portal client-credentials token source
func NewIdentity(log *util.Logger, clientID, clientSecret string) oauth2.TokenSource {
	v := &identity{
		Helper:       request.NewHelper(log),
		clientID:     clientID,
		clientSecret: clientSecret,
		uri:          BaseURL + "/token",
	}
	return oauth2.ReuseTokenSource(nil, oauth.Redacted(log, v))
}

// Token implements oauth2.TokenSource using the client-credentials grant
func (v *identity) Token() (*oauth2.Token, error) {
	data := struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
		Scope        string `json:"scope"`
	}{
		ClientID:     v.clientID,
		ClientSecret: v.clientSecret,
		Scope:        strings.Join(scopes, " "),
	}

	req, _ := request.New(http.MethodPost, v.uri, request.MarshalJSON(data), request.JSONEncoding)

	// the Data Portal uses non-standard camelCase token fields
	var res struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int64  `json:"expiresIn"`
		TokenType   string `json:"tokenType"`
	}
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken: res.AccessToken,
		TokenType:   res.TokenType,
		ExpiresIn:   res.ExpiresIn,
	}

	return util.TokenWithExpiry(token), nil
}
