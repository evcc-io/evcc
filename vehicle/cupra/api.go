package cupra

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// BaseURL is the API base url
const BaseURL = "https://ola.prod.code.seat.cloud.vwgroup.com"

// API is an api.Vehicle implementation for Seat Cupra cars
type API struct {
	*request.Helper
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, ts oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   v.Client.Transport,
	}

	return v
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles(userID string) (res []string, err error) {
	var vehicles struct {
		Vehicles []Vehicle
	}

	uri := fmt.Sprintf("%s/v1/users/%s/garage/vehicles", BaseURL, userID)
	err = v.GetJSON(uri, &vehicles)

	for _, v := range vehicles.Vehicles {
		res = append(res, v.VIN)
	}

	return res, err
}

// Status implements the /status response
func (v *API) Status(userID, vin string) (Status, error) {
	var res Status
	uri := fmt.Sprintf("%s/v2/users/%s/vehicles/%s/mycar", BaseURL, userID, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}
