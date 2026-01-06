package query

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const ApiURI = "https://api.vehicle.ford.com/fcon-query"

// API is the Ford api client
type API struct {
	*request.Helper
}

// NewAPI creates a new api client
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

// Vehicles returns the list of user vehicles
func (v *API) Vehicles() ([]Vehicle, error) {
	var res Vehicle

	uri := fmt.Sprintf("%s/v1/garage", ApiURI)
	err := v.GetJSON(uri, &res)

	return []Vehicle{res}, err
}

func (v *API) Telemetry(_ string) (Telemetry, error) {
	var res Telemetry

	uri := fmt.Sprintf("%s/v1/telemetry", ApiURI)
	err := v.GetJSON(uri, &res)

	return res, err
}
