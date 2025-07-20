package seat

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// BaseURL is the API base url
const BaseURL = "https://mal-3a.prd.eu.dp.vwg-connect.com/api"

// API provides list of vehicles for Seat cars
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
func (v *API) Vehicles(userID string) ([]string, error) {
	var res struct {
		UserVehicles struct {
			UserId  string
			Vehicle []struct {
				Content string
			}
		}
	}

	uri := fmt.Sprintf("%s/usermanagement/users/v2/users/%s/vehicles", BaseURL, userID)
	err := v.GetJSON(uri, &res)

	var vehicles []string
	for _, v := range res.UserVehicles.Vehicle {
		vehicles = append(vehicles, v.Content)
	}

	return vehicles, err
}
