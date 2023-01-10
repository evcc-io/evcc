package id

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// https://identity-userinfo.vwgroup.io/oidc/userinfo
// https://customer-profile.apps.emea.vwapps.io/v1/customers/<userId>/realCarData

// BaseURL is the API base url
const BaseURL = "https://mobileapi.apps.emea.vwapps.io"

// API is an api.Vehicle implementation for VW ID cars
type API struct {
	*request.Helper
}

// Actions and action values
const (
	ActionCharge         = "charging"
	ActionChargeStart    = "start"
	ActionChargeStop     = "stop"
	ActionChargeSettings = "settings" // body: targetSOC_pct

	ActionClimatisation      = "climatisation"
	ActionClimatisationStart = "start"
	ActionClimatisationStop  = "stop"
)

// NewAPI creates a new vehicle
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

// Vehicles implements the /vehicles response
func (v *API) Vehicles() (res []string, err error) {
	uri := fmt.Sprintf("%s/vehicles", BaseURL)

	req, err := request.New(http.MethodGet, uri, nil, request.AcceptJSON)

	var vehicles struct {
		Data []struct {
			VIN      string
			Model    string
			Nickname string
		}
	}

	if err == nil {
		err = v.DoJSON(req, &vehicles)

		for _, v := range vehicles.Data {
			res = append(res, v.VIN)
		}
	}

	return res, err
}

// Status implements the /status response.
// It is callers responsibility to check for embedded (partial) errors.
func (v *API) Status(vin string) (res Status, err error) {
	// NOTE use `all` to retrieve entire status
	uri := fmt.Sprintf("%s/vehicles/%s/selectivestatus?jobs=charging,fuelStatus,climatisation", BaseURL, vin)

	req, err := request.New(http.MethodGet, uri, nil, request.AcceptJSON)

	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

// Action implements vehicle actions
func (v *API) Action(vin, action, value string) error {
	uri := fmt.Sprintf("%s/vehicles/%s/%s/%s", BaseURL, vin, action, value)

	req, err := request.New(http.MethodPost, uri, nil, request.AcceptJSON)

	if err == nil {
		var res interface{}
		err = v.DoJSON(req, &res)
	}

	return err
}

// Any implements any api response
func (v *API) Any(uri, vin string) (interface{}, error) {
	if strings.Contains(uri, "%s") {
		uri = fmt.Sprintf(uri, vin)
	}

	req, err := request.New(http.MethodGet, uri, nil, request.AcceptJSON)

	var res interface{}
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}
