package connected

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type oauth2Config struct {
	*oauth2.Config
	h *request.Helper
}

func Oauth2Config(log *util.Logger, id, secret, redirect string) *oauth2Config {
	return &oauth2Config{
		h: request.NewHelper(log),
		Config: &oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			RedirectURL:  redirect,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://volvoid.eu.volvocars.com/as/authorization.oauth2",
				TokenURL: "https://volvoid.eu.volvocars.com/as/token.oauth2",
			},
			Scopes: []string{
				oidc.ScopeOpenID,
				"vehicle:attributes",
				"energy:recharge_status", "energy:battery_charge_level", "energy:electric_range", "energy:estimated_charging_time", "energy:charging_connection_status", "energy:charging_system_status",
				"conve:fuel_status", "conve:odometer_status", "conve:environment",
			},
		},
	}
}

func (oc *oauth2Config) TokenSource(ctx context.Context, token *oauth2.Token) oauth2.TokenSource {
	return oauth.RefreshTokenSource(token, oc)
}

func (oc *oauth2Config) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		// "access_token_manager_id": {managerId},
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
	}

	req, err := request.New(http.MethodPost, oc.Endpoint.TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": request.FormContent,
		// "Authorization": basicAuth,
	})
	if err != nil {
		return nil, err
	}

	var res oauth2.Token
	if err := oc.h.DoJSON(req, &res); err != nil {
		return nil, err
	}

	return util.TokenWithExpiry(&res), err
}
