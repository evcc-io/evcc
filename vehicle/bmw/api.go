package bmw

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// https://github.com/bimmerconnected/bimmer_connected

const ApiURI = "https://b2vapi.bmwgroup.com/webapi/v1"

type StatusResponse struct {
	VehicleStatus struct {
		ConnectionStatus       string // CONNECTED
		ChargingStatus         string // CHARGING, ERROR, FINISHED_FULLY_CHARGED, FINISHED_NOT_FULL, INVALID, NOT_CHARGING, WAITING_FOR_CHARGING
		ChargingLevelHv        int
		RemainingRangeElectric int
		Mileage                int
		// UpdateTime             time.Time // 2021-08-12T12:00:08+0000
	}
}

type VehiclesResponse struct {
	Vehicles []Vehicle
}

type Vehicle struct {
	VIN   string
	Model string
}

// API is an api.Vehicle implementation for BMW cars
type API struct {
	*request.Helper
}

// NewAPI creates a new vehicle
func NewAPI(log *util.Logger, identity oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
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
func (v *API) Status(vin string) (StatusResponse, error) {
	var resp StatusResponse
	uri := fmt.Sprintf("%s/user/vehicles/%s/status", ApiURI, vin)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	return resp, err
}
