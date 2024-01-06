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
	"response_type": {"code id_token"}, // token
	"client_id":     {"3c8e98bc-3ae9-4277-a563-d5ee65ddebba@apps_vw-dilab_com"},
	"redirect_uri":  {"seatconnect://identity-kit/login"},
	"scope":         {"openid profile"}, // address phone email birthdate nationalIdentifier cars mbb dealers badge nationality
}
