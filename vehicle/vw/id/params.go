package id

import (
	"net/url"

	"github.com/evcc-io/evcc/vehicle/vag/loginapps"
)

const LoginURL = loginapps.BaseURL + "/v1/authorize"

var AuthParams = url.Values{
	"response_type": {"code id_token token"},
	"client_id":     {"a24fba63-34b3-4d43-b181-942111e6bda8@apps_vw-dilab_com"},
	"redirect_uri":  {"weconnect://authenticated"},
	"scope":         {"openid profile badge cars vin"}, // dealers
}
