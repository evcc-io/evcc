package tronity

import (
	"golang.org/x/oauth2"
)

const URI = "https://api-eu.tronity.io"

// OAuth2Config is the OAuth2 configuration for authenticating with the Tesla API.
var OAuth2Config = &oauth2.Config{
	ClientID:    "6f636cb0-1807-4e74-b522-dc2a6a8db78f", // evcc
	RedirectURL: "http://localhost:8080",
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://api-eu.tronity.io/oauth/authorize",
		TokenURL: "https://api-eu.tronity.io/oauth/authentication",
	},
	Scopes: []string{"read_vin", "read_vehicle_info", "read_odometer", "read_charge", "read_charge", "read_battery", "read_location", "write_charge_start_stop", "write_wake_up"},
}

// func OAuth2Config(id, secret string) *oauth2.Config {
// 	c := *oAuth2Config
// 	if err := mergo.Merge(&c, &oauth2.Config{
// 		ClientID:     id,
// 		ClientSecret: secret,
// 	}, mergo.WithOverride); err != nil {
// 		panic(err)
// 	}

// 	return &c
// }

// type tokenProxy struct {
// 	oauth2.TokenSource
// }

// func (t *tokenProxy) Token() (*oauth2.Token, error) {
// 	token, err := t.TokenSource.Token()
// 	if err == nil {
// 		token.TokenType = "Bearer"
// 	}
// 	return token, err
// }

// func TokenSource(ctx context.Context, token *oauth2.Token) oauth2.TokenSource {
// 	ts := OAuth2Config.TokenSource(ctx, token)
// 	return &tokenProxy{ts}
// }
