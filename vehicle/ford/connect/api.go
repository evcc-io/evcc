package connect

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

const ApiURI = "https://api.mps.ford.com/api/fordconnect"

// API is the Ford api client
type API struct {
	*request.Helper
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, ts oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &transport.Decorator{
		Base: &oauth2.Transport{
			Source: ts,
			Base:   v.Client.Transport,
		},
		Decorator: transport.DecorateHeaders(map[string]string{
			"Application-Id": ApplicationID,
		}),
	}

	return v
}

// Vehicles returns the list of user vehicles
func (v *API) Vehicles() ([]Vehicle, error) {
	var res VehiclesResponse

	uri := fmt.Sprintf("%s/v3/vehicles", ApiURI)
	err := v.GetJSON(uri, &res)

	return res.Vehicles, err
}

// VIN returns the vehicle's vIN
func (v *API) VIN(id string) (string, error) {
	var res struct {
		VIN string
	}

	uri := fmt.Sprintf("%s/v3/vehicles/%s/vin", ApiURI, id)
	err := v.GetJSON(uri, &res)

	return res.VIN, err
}

func (v *API) Status(vin string) (Vehicle, error) {
	var res InformationResponse

	uri := fmt.Sprintf("%s/v3/vehicles/%s", ApiURI, vin)
	err := v.GetJSON(uri, &res)

	if err == nil && res.Status != StatusSuccess {
		err = fmt.Errorf("status %s", res.Status)
	}

	return res.Vehicle, err
}
