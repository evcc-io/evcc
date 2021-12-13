package bmw

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/logx"
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
func NewAPI(log logx.Logger, brand string, identity oauth2.TokenSource) *API {
	v := &API{
		Helper:     request.NewHelper(log),
		xUserAgent: fmt.Sprintf("android(v1.07_20200330);%s;1.7.0(11152)", brand),
	}

	// replace client transport with authenticated transport
	v.Client.Transport = &oauth2.Transport{
		Source: identity,
		Base:   v.Client.Transport,
	}

	return v
}

// Vehicles implements returns the /user/vehicles api
func (v *API) Vehicles() ([]string, error) {
	var resp VehiclesResponse
	uri := fmt.Sprintf("%s/user/vehicles", ApiURI)
	// uri := fmt.Sprintf("%s/eadrax-vcs/v1/vehicles", CocoApiURI, vin)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	var vehicles []string
	for _, v := range resp.Vehicles {
		vehicles = append(vehicles, v.VIN)
	}

	return vehicles, err
}

// Status implements the /user/vehicles/<vin>/status api
func (v *API) Status(vin string) (VehicleStatus, error) {
	var resp VehiclesStatusResponse
	uri := fmt.Sprintf("%s/eadrax-vcs/v1/vehicles?apptimezone=60&appDateTime=%d", CocoApiURI, time.Now().Unix())

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"X-User-Agent": v.xUserAgent,
	})
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

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
