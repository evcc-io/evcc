package tronity

import (
	"golang.org/x/oauth2"
)

const URI = "https://api.tronity.tech"

func OAuth2Config(id, secret string) (*oauth2.Config, error) {
	return &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.tronity.io/oauth/v2/authorize",
			TokenURL: "https://api.tronity.tech/authentication",
		},
		Scopes: []string{"read_vin", "read_vehicle_info", "read_odometer", "read_charge", "read_charge", "read_battery", "read_location", "write_charge_start_stop", "write_wake_up"},
	}, nil
}
