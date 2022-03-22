package idkproxy

import "golang.org/x/oauth2"

const WellKnown = "https://idkproxy-service.apps.emea.vwapps.io/v1/emea/openid-configuration"

var Endpoint = &oauth2.Endpoint{
	AuthURL:  "https://identity.vwgroup.io/oidc/v1/authorize",
	TokenURL: "https://idkproxy-service.apps.emea.vwapps.io/v1/emea/token",
}
