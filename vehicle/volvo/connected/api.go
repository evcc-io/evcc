package connected

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
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
func NewAPI(log *util.Logger, identity oauth2.TokenSource, vccapikey string) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	// replace client transport with authenticated transport
	v.Client.Transport = &oauth2.Transport{
		Source: identity,
		Base: &transport.Decorator{
			Base: v.Client.Transport,
			Decorator: transport.DecorateHeaders(map[string]string{
				"vcc-api-key": vccapikey,
			}),
		},
	}

	return v
}

func (v *API) Vehicles() ([]string, error) {
	type Vehicle struct {
		ID string
	}
	var res struct {
		Vehicles []Vehicle
	}

	uri := fmt.Sprintf("%s/extended-vehicle/v1/vehicles", ApiURL)
	err := v.GetJSON(uri, &res)

	return lo.Map(res.Vehicles, func(v Vehicle, _ int) string {
		return v.ID
	}), err
}

// Range provides range status api response
func (v *API) RechargeStatus(vin string) (RechargeStatus, error) {
	uri := fmt.Sprintf("%s/energy/v1/vehicles/%s/recharge-status", ApiURL, vin)
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept": "application/vnd.volvocars.api.energy.vehicledata.v1+json",
	})

	var res RechargeStatus
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}
