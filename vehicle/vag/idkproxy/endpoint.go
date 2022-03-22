package idkproxy

import (
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"golang.org/x/oauth2"
)

const (
	BaseURL   = "https://idkproxy-service.apps.emea.vwapps.io"
	WellKnown = BaseURL + "/v1/emea/openid-configuration"
)

var Endpoint = &oauth2.Endpoint{
	AuthURL:  vwidentity.Endpoint.AuthURL,
	TokenURL: BaseURL + "/v1/emea/token",
}
