package cupra

import "net/url"

// Authorization parameters
var AuthParams = url.Values{
	"response_type": {"code id_token"}, // token
	// "client_id":     {"30e33736-c537-4c72-ab60-74a7b92cfe83@apps_vw-dilab_com"},
	// "redirect_uri":  {"cupraconnect://identity-kit/login"},
	"client_id":     {"3c756d46-f1ba-4d78-9f9a-cff0d5292d51@apps_vw-dilab_com"},
	"client_secret": {"eb8814e641c81a2640ad62eeccec11c98effc9bccd4269ab7af338b50a94b3a2"},
	"redirect_uri":  {"cupra://oauth-callback"},
	"scope":         {"openid profile mbb"}, // cars birthdate nickname address phone
}
