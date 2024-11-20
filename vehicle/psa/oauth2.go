package psa

import (
	"fmt"

	"golang.org/x/oauth2"
)

func Oauth2Config(brand, country string) *oauth2.Config {
	switch brand {
	case "citroen":
		return &oauth2.Config{
			ClientID:     "5364defc-80e6-447b-bec6-4af8d1542cae",
			ClientSecret: "iE0cD8bB0yJ0dS6rO3nN1hI2wU7uA5xR4gP7lD6vM0oH0nS8dN",
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://idpcvs.citroen.com/am/oauth2/authorize",
				TokenURL:  "https://idpcvs.citroen.com/am/oauth2/access_token",
				AuthStyle: oauth2.AuthStyleInHeader,
			},
			RedirectURL: fmt.Sprintf("mymacsdk://oauth2redirect/%s", country),
			Scopes:      []string{"openid", "profile"},
		}

	case "ds":
		return &oauth2.Config{
			ClientID:     "cbf74ee7-a303-4c3d-aba3-29f5994e2dfa",
			ClientSecret: "X6bE6yQ3tH1cG5oA6aW4fS6hK0cR0aK5yN2wE4hP8vL8oW5gU3",
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://idpcvs.driveds.com/am/oauth2/authorize",
				TokenURL:  "https://idpcvs.driveds.com/am/oauth2/access_token",
				AuthStyle: oauth2.AuthStyleInHeader,
			},
			RedirectURL: fmt.Sprintf("mymdssdk://oauth2redirect/%s", country),
			Scopes:      []string{"openid", "profile"},
		}

	case "opel":
		return &oauth2.Config{
			ClientID:     "07364655-93cb-4194-8158-6b035ac2c24c",
			ClientSecret: "F2kK7lC5kF5qN7tM0wT8kE3cW1dP0wC5pI6vC0sQ5iP5cN8cJ8",
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://idpcvs.opel.com/am/oauth2/authorize",
				TokenURL:  "https://idpcvs.opel.com/am/oauth2/access_token",
				AuthStyle: oauth2.AuthStyleInHeader,
			},
			RedirectURL: fmt.Sprintf("mymopsdk://oauth2redirect/%s", country),
			Scopes:      []string{"openid", "profile"},
		}

	case "peugeot":
		return &oauth2.Config{
			ClientID:     "1eebc2d5-5df3-459b-a624-20abfcf82530",
			ClientSecret: "T5tP7iS0cO8sC0lA2iE2aR7gK6uE5rF3lJ8pC3nO1pR7tL8vU1",
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://idpcvs.peugeot.com/am/oauth2/authorize",
				TokenURL:  "https://idpcvs.peugeot.com/am/oauth2/access_token",
				AuthStyle: oauth2.AuthStyleInHeader,
			},
			RedirectURL: fmt.Sprintf("mymap://oauth2redirect/%s", country),
			Scopes:      []string{"openid", "profile"},
		}

	default:
		panic("invalid brand: " + brand)
	}
}
