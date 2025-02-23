package skoda

import "net/url"

const (
	Brand   = "VW"
	Country = "CZ"

	// Authorization ClientID
	AuthClientID = "afb0473b-6d82-42b8-bfea-cead338c46ef"
)

// Skoda native api
var AuthParams = url.Values{
	"response_type": {"code id_token"},
	"client_id":     {"f9a2359a-b776-46d9-bd0c-db1904343117@apps_vw-dilab_com"},
	// "redirect_uri":  {"skodaconnect://oidc.login/"}, // old
	"redirect_uri": {"myskoda://redirect/login/"}, // new
	"scope":        {"openid mbb profile"},
}

// TokenRefreshService parameters
var TRSParams = url.Values{
	"brand": {"skoda"},
}
