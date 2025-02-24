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
	"client_id":     {"7f045eee-7003-4379-9968-9355ed2adb06@apps_vw-dilab_com"},
	"redirect_uri":  {"myskoda://redirect/login/"},
	"scope":         {"address badge birthdate cars driversLicense dealers email mileage mbb nationalIdentifier openid phone profession profile vin"},
}

// TokenRefreshService parameters
var TRSParams = url.Values{
	"brand": {"skoda"},
}
