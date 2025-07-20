package psa

import (
	"fmt"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

// https://developer.groupe-psa.io/webapi/b2c/api-reference/specification

// BaseURL is the API base url
const BaseURL = "https://api.groupe-psa.com/connectedcar/v4"

// API is an api.Vehicle implementation for PSA cars
type API struct {
	*request.Helper
	realm string
	id    string
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity oauth2.TokenSource, realm, id string) *API {
	v := &API{
		Helper: request.NewHelper(log),
		realm:  realm,
		id:     id,
	}

	// replace client transport with authenticated transport plus headers
	v.Client.Transport = &transport.Decorator{
		Base: &oauth2.Transport{
			Source: identity,
			Base:   v.Client.Transport,
		},
		Decorator: transport.DecorateHeaders(map[string]string{
			"Accept":             "application/hal+json",
			"X-Introspect-Realm": v.realm,
		}),
	}

	return v
}

func (v *API) clientID() string {
	return url.Values{"client_id": {v.id}}.Encode()
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles() ([]Vehicle, error) {
	var res struct {
		Embedded struct {
			Vehicles []Vehicle
		} `json:"_embedded"`
	}

	uri := fmt.Sprintf("%s/user/vehicles?%s", BaseURL, v.clientID())
	err := v.GetJSON(uri, &res)

	return res.Embedded.Vehicles, err
}

// Status implements the /vehicles/<vid>/status response
func (v *API) Status(vid string) (Status, error) {
	var res Status

	uri := fmt.Sprintf("%s/user/vehicles/%s/status?%s", BaseURL, vid, v.clientID())
	err := v.GetJSON(uri, &res)

	return res, err
}
