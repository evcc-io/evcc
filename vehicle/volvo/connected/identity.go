package connected

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

var Oauth2Config = oauth2.Config{
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://volvoid.eu.volvocars.com/as/authorization.oauth2",
		TokenURL: "https://volvoid.eu.volvocars.com/as/token.oauth2",
	},
	Scopes: []string{oidc.ScopeOpenID, "vehicle:attributes",
		"energy:recharge_status", "energy:battery_charge_level", "energy:electric_range", "energy:estimated_charging_time", "energy:charging_connection_status", "energy:charging_system_status",
		"conve:fuel_status", "conve:odometer_status", "conve:environment",
	},
}

const (
	managerId = "JWTh4Yf0b"
	basicAuth = "Basic aDRZZjBiOlU4WWtTYlZsNnh3c2c1WVFxWmZyZ1ZtSWFEcGhPc3kxUENhVXNpY1F0bzNUUjVrd2FKc2U0QVpkZ2ZJZmNMeXc="
)

type Identity struct {
	*request.Helper
	oauth2.TokenSource
}

func NewIdentity(log *util.Logger) (*Identity, error) {
	v := &Identity{
		Helper: request.NewHelper(log),
	}

	return v, nil
}

func (v *Identity) Login(user, password string) error {
	data := url.Values{
		"username":                {user},
		"password":                {password},
		"access_token_manager_id": {managerId},
		"grant_type":              {"password"},
		"scope":                   {strings.Join(Oauth2Config.Scopes, " ")},
	}

	req, err := request.New(http.MethodPost, Oauth2Config.Endpoint.TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type":  request.FormContent,
		"Authorization": basicAuth,
	})

	if err == nil {
		var token oauth2.Token
		if err = v.DoJSON(req, &token); err == nil {
			v.TokenSource = oauth.RefreshTokenSource(&token, v)
		}
	}

	return err
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"access_token_manager_id": {managerId},
		"grant_type":              {"refresh_token"},
		"refresh_token":           {token.RefreshToken},
	}

	req, err := request.New(http.MethodPost, Oauth2Config.Endpoint.TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type":  request.FormContent,
		"Authorization": basicAuth,
	})

	var res oauth2.Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}
