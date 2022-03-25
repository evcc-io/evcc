package cupra

import "net/url"

// TODO check AuthParam usage (currently using seat)

// Authorization parameters
var AuthParams = url.Values{
	"response_type": {"code id_token"},
	"client_id":     {"3c756d46-f1ba-4d78-9f9a-cff0d5292d51@apps_vw-dilab_com"},
	"redirect_uri":  {"cupra://oauth-callback"},
	"scope":         {"openid profile mbb"}, // cars birthdate nickname address phone
}
