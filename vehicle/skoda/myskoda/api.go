package myskoda

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const BaseURI = "https://mysmob.api.connect.skoda-auto.cz/api"

// API is the Skoda api client
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

// Vehicles implements the /v2/garage response
func (v *API) Vehicles() ([]Vehicle, error) {
	var res VehiclesResponse

	uri := fmt.Sprintf("%s/v2/garage", BaseURI)
	err := v.GetJSON(uri, &res)

	return res.Vehicles, err
}

func (v *API) VehicleDetails(vin string) (Vehicle, error) {
	var res Vehicle
	uri := fmt.Sprintf("%s/v2/garage/vehicles/%s", BaseURI, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}

// Status implements the /v2/vehicle-status/<vin> response
func (v *API) Status(vin string) (StatusResponse, error) {
	var res StatusResponse
	uri := fmt.Sprintf("%s/v1/vehicle-health-report/warning-lights/%s", BaseURI, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}

// Charger implements the /v1/charging/<vin> response
func (v *API) Charger(vin string) (ChargerResponse, error) {
	var res ChargerResponse
	uri := fmt.Sprintf("%s/v1/charging/%s", BaseURI, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}

// Settings implements the /v1/charging/<vin>/settings response
func (v *API) Settings(vin string) (SettingsResponse, error) {
	var res SettingsResponse

	chrgRes, err := v.Charger(vin)
	if err == nil {
		res = chrgRes.Settings
	}
	return res, err
}

// Climater implements the /v2/air-conditioning/<vin> response
func (v *API) Climater(vin string) (ClimaterResponse, error) {
	var res ClimaterResponse
	uri := fmt.Sprintf("%s/v2/air-conditioning/%s", BaseURI, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}

const (
	ActionCharge      = "charging"
	ActionChargeStart = "start"
	ActionChargeStop  = "stop"
)

// Action executes a vehicle action
func (v *API) Action(vin, action, value string) error {
	// @POST("api/v1/charging/{vin}/start")
	// @POST("api/v1/charging/{vin}/stop")
	uri := fmt.Sprintf("%s/v1/%s/%s/%s", BaseURI, action, vin, value)

	req, err := request.New(http.MethodPost, uri, nil, request.JSONEncoding)
	if err == nil {
		err = v.DoJSON(req, nil)
	}
	return err
}

func (v *API) WakeUp(vin string) error {
	// @POST("api/v1/vehicle-wakeup/{vin}")
	uri := fmt.Sprintf("%s/v1/vehicle-wakeup/%s", BaseURI, vin)

	req, err := request.New(http.MethodPost, uri, nil, request.JSONEncoding)
	if err == nil {
		err = v.DoJSON(req, nil)
	}
	return err
}
