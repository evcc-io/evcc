package skoda

import "net/url"

const (
	Brand   = "VW"
	Country = "CZ"

	// Authorization ClientID
	AuthClientID = "afb0473b-6d82-42b8-bfea-cead338c46ef"
)

// Skoda native api
var AuthParams = url.Values(map[string][]string{
	"response_type": {"code id_token"},
	"client_id":     {"f9a2359a-b776-46d9-bd0c-db1904343117@apps_vw-dilab_com"},
	"redirect_uri":  {"skodaconnect://oidc.login/"},
	"scope":         {"openid mbb profile"},
})

// Skoda connect api
var ConnectAuthParams = url.Values(map[string][]string{
	"response_type": {"code id_token"},
	"redirect_uri":  {"skodaconnect://oidc.login/"},
	"client_id":     {"7f045eee-7003-4379-9968-9355ed2adb06@apps_vw-dilab_com"},
	"scope":         {"openid profile mbb"}, // phone address cars email birthdate badge dealers driversLicense
})
