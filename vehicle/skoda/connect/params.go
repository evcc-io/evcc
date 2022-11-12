package connect

import (
	"net/url"
)

// Skoda connect api
var AuthParams = url.Values{
	"response_type": {"code id_token"},
	"redirect_uri":  {"skodaconnect://oidc.login/"},
	"client_id":     {"7f045eee-7003-4379-9968-9355ed2adb06@apps_vw-dilab_com"},
	"scope":         {"openid profile mbb mileage"}, // phone address cars email birthdate badge dealers driversLicense
}
