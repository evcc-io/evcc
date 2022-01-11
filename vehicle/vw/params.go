package vw

import "net/url"

const (
	Brand   = "VW"
	Country = "DE"

	// Authorization ClientID
	AuthClientID = "38761134-34d0-41f3-9a73-c4be88d7d337"
)

// Authorization parameters
var AuthParams = url.Values(map[string][]string{
	"response_type": {"id_token token"},
	"client_id":     {"9496332b-ea03-4091-a224-8c746b885068@apps_vw-dilab_com"},
	"redirect_uri":  {"carnet://identity-kit/login"},
	"scope":         {"openid profile mbb"}, // cars birthdate nickname address phone
})
