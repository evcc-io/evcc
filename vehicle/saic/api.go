package saic

import (
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/saic/requests"
	"golang.org/x/oauth2"
)

// API is an api.Vehicle implementation for SAIC cars
type API struct {
	identity oauth2.TokenSource
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity oauth2.TokenSource) *API {
	v := &API{
		identity: identity,
	}

	return v
}

/* Vehicles implements returns the /user/vehicles api
func (v *API) Vehicles() ([]Vehicle, error) {
	var res []Vehicle
	uri := fmt.Sprintf("%s/eadrax-vcs/v4/vehicles?apptimezone=120&appDateTime=%d", regions[v.region].CocoApiURI, time.Now().UnixMilli())
	err := v.GetJSON(uri, &res)
	return res, err
}
*/

// Status implements the /user/vehicles/<vin>/status api
func (v *API) Status(vin string) (requests.ChargeStatus, error) {
	var res requests.ChargeStatus
	answer := requests.Answer{
		Data: &res,
	}

	vinHash := requests.Sha256(vin)

	token, err := v.identity.Token()
	if err != nil {
		return res, err
	}

	// get charging status of 1st vehicle
	header, err := requests.SendRequest(requests.BASE_URL_P+"vehicle/charging/mgmtData?vin="+vinHash,
		http.MethodGet,
		"",
		"application/json",
		token.AccessToken,
		"",
		&answer)
	if err != nil {
		return res, err
	}

	_, err = requests.SendRequest("https://gateway-mg-eu.soimt.com/api.app/v1/vehicle/charging/mgmtData?vin="+vinHash,
		http.MethodGet,
		"",
		"application/json",
		token.AccessToken,
		header.Get("event-id"),
		&answer)

	return res, err
}
