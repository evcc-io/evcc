package bmw

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// https://github.com/bimmerconnected/bimmer_connected
// https://github.com/TA2k/ioBroker.bmw

const (
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
		xUserAgent: fmt.Sprintf("android(SP1A.210812.016.C1);%s;99.0.0(99999);row", brand),
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
	var res []Vehicle
	uri := fmt.Sprintf("%s/eadrax-vcs/v4/vehicles?apptimezone=120&appDateTime=%d", CocoApiURI, time.Now().UnixMilli())

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Content-Type": request.JSONContent,
		"X-User-Agent": v.xUserAgent,
	})
	if err != nil {
		return nil, err
	}

	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	var vehicles []string
	for _, v := range res {
		vehicles = append(vehicles, v.VIN)
	}

	return vehicles, err
}

// Status implements the /user/vehicles/<vin>/status api
func (v *API) Status(vin string) (VehicleStatus, error) {
	var res VehicleStatus
	uri := fmt.Sprintf("%s/eadrax-vcs/v4/vehicles/state?apptimezone=120&appDateTime=%d", CocoApiURI, time.Now().UnixMilli())

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Content-Type": request.JSONContent,
		"X-User-Agent": v.xUserAgent,
		"bmw-vin":      vin,
	})
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}
