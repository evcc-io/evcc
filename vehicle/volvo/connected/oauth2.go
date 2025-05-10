package connected

import (
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func Oauth2Config(id, secret, redirecturi string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		RedirectURL:  redirecturi,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://volvoid.eu.volvocars.com/as/authorization.oauth2",
			TokenURL:  "https://volvoid.eu.volvocars.com/as/token.oauth2",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		Scopes: []string{
			oidc.ScopeOpenID,
			"conve:vehicle_relation",
			"energy:recharge_status", "energy:battery_charge_level", "energy:electric_range", "energy:estimated_charging_time", "energy:charging_connection_status", "energy:charging_system_status",
		},
	}
}
