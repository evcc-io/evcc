package cupra

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	// BaseURL is the API base url
	BaseURL = "https://ola.prod.code.seat.cloud.vwgroup.com"

	ActionCharge      = "charging"
	ActionChargeStart = "start"
	ActionChargeStop  = "stop"
)

// API is an api.Vehicle implementation for Seat Cupra cars
type API struct {
	*request.Helper
}

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
func (v *API) Vehicles(userID string) ([]Vehicle, error) {
	var res struct {
		Vehicles []Vehicle
	}

	uri := fmt.Sprintf("%s/v2/users/%s/garage/vehicles", BaseURL, userID)
	err := v.GetJSON(uri, &res)

	return res.Vehicles, err
}

// Status implements the /status response
func (v *API) Status(userID, vin string) (Status, error) {
	var res Status
	uri := fmt.Sprintf("%s/v5/users/%s/vehicles/%s/mycar", BaseURL, userID, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}

// ParkingPosition implements the /parkingposition response
func (v *API) ParkingPosition(vin string) (Position, error) {
	var res Position
	uri := fmt.Sprintf("%s/v1/vehicles/%s/parkingposition", BaseURL, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}

// Mileage implements the /mileage response
func (v *API) Mileage(vin string) (Mileage, error) {
	var res Mileage
	uri := fmt.Sprintf("%s/v1/vehicles/%s/mileage", BaseURL, vin)
	err := v.GetJSON(uri, &res)
	return res, err
}

// Action implements the /requests response
func (v *API) Action(vin, action, cmd string) error {
	uri := fmt.Sprintf("%s/vehicles/%s/%s/requests/%s", BaseURL, vin, action, cmd)
	req, err := request.New(http.MethodPost, uri, nil, request.JSONEncoding)
	if err == nil {
		_, err = v.DoBody(req)
	}
	return err
}
