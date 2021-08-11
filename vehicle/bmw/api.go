package bmw

import (
	"fmt"
	"net/http"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

// https://github.com/bimmerconnected/bimmer_connected

const ApiURI = "https://www.bmw-connecteddrive.com/api"

type DynamicResponse struct {
	AttributesMap struct {
		ChargingHVStatus       string  `json:"chargingHVStatus"` // CHARGING, ERROR, FINISHED_FULLY_CHARGED, FINISHED_NOT_FULL, INVALID, NOT_CHARGING, WAITING_FOR_CHARGING
		ChargingLevelHv        float64 `json:"chargingLevelHv,string"`
		BERemainingRangeFuelKm float64 `json:"beRemainingRangeFuelKm,string"`
		Mileage                float64 `json:"mileage,string"`
	}
}

type VehiclesResponse []struct {
	VIN string `json:"vin"`
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

// Vehicles implements returns the /me/vehicles api
func (v *API) Vehicles() ([]string, error) {
	var resp VehiclesResponse
	uri := fmt.Sprintf("%s/me/vehicles/v2/", ApiURI)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	var vehicles []string
	for _, v := range resp {
		vehicles = append(vehicles, v.VIN)
	}

	return vehicles, err
}

// Dynamic implements the /vehicle/dynamic api
func (v *API) Dynamic(vin string) (DynamicResponse, error) {
	var resp DynamicResponse
	uri := fmt.Sprintf("%s/vehicle/dynamic/v1/%s", ApiURI, vin)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	return resp, err
}
