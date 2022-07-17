package bmw

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// https://github.com/bimmerconnected/bimmer_connected
// https://github.com/TA2k/ioBroker.bmw

const (
	ApiURI     = "https://b2vapi.bmwgroup.com/webapi/v1"
	CocoApiURI = "https://cocoapi.bmwgroup.com"
)

// API is an api.Vehicle implementation for BMW cars
type API struct {
	*request.Helper
	xUserAgent string
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, brand string, identity oauth2.TokenSource) *API {
	v := &API{
		Helper:     request.NewHelper(log),
		xUserAgent: fmt.Sprintf("android(SP1A.210812.016.C1);%s;2.5.2(14945);row", brand),
	}

	// replace client transport with authenticated transport
	v.Client.Transport = &oauth2.Transport{
		Source: identity,
		Base:   v.Client.Transport,
	}

	return v
}

func (v *API) eadrax() (VehiclesStatusResponse, error) {
	var res VehiclesStatusResponse
	uri := fmt.Sprintf("%s/eadrax-vcs/v1/vehicles?apptimezone=120&appDateTime=%d", CocoApiURI, time.Now().Unix())

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Content-Type":          request.JSONContent,
		"X-User-Agent":          v.xUserAgent,
		"bmw-units-preferences": "d=KM;v=L",
	})
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

// Vehicles implements returns the /user/vehicles api
func (v *API) Vehicles() ([]string, error) {
	resp, err := v.eadrax()
	if err != nil {
		return nil, err
	}

	var vehicles []string
	for _, v := range resp {
		vehicles = append(vehicles, v.VIN)
	}

	return vehicles, err
}

// Status implements the /user/vehicles/<vin>/status api
func (v *API) Status(vin string) (VehicleStatus, error) {
	resp, err := v.eadrax()
	if err == nil {
		for _, res := range resp {
			if res.VIN == vin {
				return res, nil
			}
		}

		err = api.ErrNotAvailable
	}

	return VehicleStatus{}, err
}
