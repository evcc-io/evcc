package cupra

import "net/url"

// Authorization parameters
var AuthParams = url.Values{
	"response_type": {"code id_token"}, // token
	"client_id":     {"30e33736-c537-4c72-ab60-74a7b92cfe83@apps_vw-dilab_com"},
	"redirect_uri":  {"cupraconnect://identity-kit/login"},
	"scope":         {"openid profile mbb"}, // cars birthdate nickname address phone
}

// TokenRefreshService parameters
var TRSParams = url.Values{
	"brand": {"cupra"},
}
