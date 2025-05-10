package connected

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
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
func NewAPI(log *util.Logger, vccapikey string, authorizer auth.Authorizer) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	decoratedTransport := &transport.Decorator{
		Base: v.Client.Transport,
		Decorator: transport.DecorateHeaders(map[string]string{
			"vcc-api-key": vccapikey,
		}),
	}
	v.Client.Transport = authorizer.Transport(decoratedTransport)

	return v
}

func (v *API) Vehicles() ([]string, error) {
	var res struct {
		Vehicles []Vehicle `json:"data"`
	}

	uri := fmt.Sprintf("%s/connected-vehicle/v2/vehicles", ApiURL)
	err := v.GetJSON(uri, &res)

	return lo.Map(res.Vehicles, func(v Vehicle, _ int) string {
		return v.VIN
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
