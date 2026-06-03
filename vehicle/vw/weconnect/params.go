package weconnect

import (
	"net/url"

	"github.com/evcc-io/evcc/vehicle/vag/cariad"
)

// AuthParams are the OIDC authorize parameters for the WeConnect ID client.
// The authorize endpoint is the legacy identity.vwgroup.io OIDC entry point
// (vwidentity.Config.AuthURL); the token endpoint is the cariad BFF (see
// oauth.go).
var AuthParams = url.Values{
	"response_type": {"code id_token token"},
	"client_id":     {cariad.ClientID},
	"redirect_uri":  {"weconnect://authenticated"},
	"scope":         {"openid profile badge cars vin"}, // dealers
}
