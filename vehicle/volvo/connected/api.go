package connected

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

// api constants
const (
	ApiURL = "https://api.volvocars.com"
)

// API is the Volvo client
type API struct {
	*request.Helper
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, vccapikey string, ts oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base: &transport.Decorator{
			Decorator: transport.DecorateHeaders(map[string]string{
				"vcc-api-key": vccapikey,
			}),
			Base: v.Client.Transport,
		},
	}

	return v
}

// Range provides range status api response
func (v *API) EnergyState(vin string) (EnergyState, error) {
	uri := fmt.Sprintf("%s/energy/v2/vehicles/%s/state", ApiURL, vin)

	var res EnergyState
	err := v.GetJSON(uri, &res)

	return res, err
}

// Range provides range status api response
func (v *API) OdometerState(vin string) (OdometerState, error) {
	uri := fmt.Sprintf("%s/connected-vehicle/v2/vehicles/%s/odometer", ApiURL, vin)

	var res OdometerState
	err := v.GetJSON(uri, &res)

	return res, err
}
