package seat

import "net/url"

const (
	Brand   = "VW"
	Country = "ES"

	// Authorization ClientID
	AuthClientID = "9dcc70f0-8e79-423a-a3fa-4065d99088b4"
)

// Authorization parameters
var AuthParams = url.Values{
	"response_type": {"code id_token"},
	"client_id":     {"50f215ac-4444-4230-9fb1-fe15cd1a9bcc@apps_vw-dilab_com"},
	"redirect_uri":  {"seatconnect://identity-kit/login"},
	"scope":         {"openid profile mbb"}, // cars birthdate nickname address phone
}
