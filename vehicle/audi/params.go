package audi

import "net/url"

// Authorization parameters
var AuthParams = url.Values{
	"response_type": {"code"},
	"client_id":     {"f4d0934f-32bf-4ce4-b3c4-699a7049ad26@apps_vw-dilab_com"},
	"redirect_uri":  {"myaudi:///"},
	"scope":         {"openid profile mbb"}, // vin badge birthdate nickname email address phone name picture
	"prompt":        {"login"},
	"ui_locales":    {"de-DE"},
}

var IDKParams = url.Values{
	"client_id":    {"f4d0934f-32bf-4ce4-b3c4-699a7049ad26@apps_vw-dilab_com"},
	"redirect_uri": {"myaudi:///"},
}

const AZSConfig = "myaudi"
