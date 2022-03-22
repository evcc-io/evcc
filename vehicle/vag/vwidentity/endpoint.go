package vwidentity

import "golang.org/x/oauth2"

const WellKnown = "https://identity.vwgroup.io/.well-known/openid-configuration"

var Endpoint = &oauth2.Endpoint{
	AuthURL:  "https://identity.vwgroup.io/oidc/v1/authorize",
	TokenURL: "https://identity.vwgroup.io/oidc/v1/token",
}
