package seat

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	// BaseURL is the API base url
	BaseURL = "https://mal-3a.prd.eu.dp.vwg-connect.com/api"
)

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

type Vehicle struct {
	VIN string
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles(userID string) ([]Vehicle, error) {
	var res struct {
		Vehicles []Vehicle
	}

	uri := fmt.Sprintf("%s/usermanagement/users/v2/users/%s/vehicles", BaseURL, userID)
	err := v.GetJSON(uri, &res)

	// return res.Vehicles, err
	return nil, err
}
