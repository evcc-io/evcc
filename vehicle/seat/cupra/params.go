package cupra

import (
	"golang.org/x/oauth2"
)

var OAuth2Config = &oauth2.Config{
	ClientID:     "3c756d46-f1ba-4d78-9f9a-cff0d5292d51@apps_vw-dilab_com",
	ClientSecret: "eb8814e641c81a2640ad62eeccec11c98effc9bccd4269ab7af338b50a94b3a2",
	RedirectURL:  "cupra://oauth-callback",
	Scopes:       []string{"openid", "profile", "mbb"},
}
