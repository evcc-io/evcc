package audi

import "net/url"

const (
	Brand   = "Audi"
	Country = "DE"

	// Authorization ClientID
	AuthClientID = "77869e21-e30a-4a92-b016-48ab7d3db1d8"
)

// Authorization parameters
var AuthParams = url.Values{
	"response_type": {"id_token token"},
	"client_id":     {"09b6cbec-cd19-4589-82fd-363dfa8c24da@apps_vw-dilab_com"},
	"redirect_uri":  {"myaudi:///"},
	"scope":         {"openid profile mbb"}, // vin badge birthdate nickname email address phone name picture
	"prompt":        {"login"},
	"ui_locales":    {"de-DE"},
}
