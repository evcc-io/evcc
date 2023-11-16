package vc

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bogosj/tesla"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	FleetAudienceEU = "https://fleet-api.prd.eu.vn.cloud.tesla.com/"
)

type API struct {
	*request.Helper
	identity *Identity
}

func NewAPI(log *util.Logger, identity *Identity) *API {
	client := request.NewHelper(log)
	client.Transport = &oauth2.Transport{
		Source: identity,
		Base:   client.Transport,
	}

	return &API{
		Helper:   client,
		identity: identity,
	}
}

func (v *API) Vehicles() ([]*Vehicle, error) {
	// ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	// defer cancel()

	// b, err := v.identity.Account().Get(ctx, "api/1/vehicles")
	// if err != nil {
	// 	return nil, err
	// }

	// var res tesla.VehiclesResponse
	// if err := json.Unmarshal(b, &res); err != nil {
	// 	return nil, err
	// }

	var res tesla.VehiclesResponse
	err := v.GetJSON(fmt.Sprintf("%sapi/1/vehicles", FleetAudienceEU), &res)

	return res.Response, err
}

func (v *API) VehicleData(id int64) (*VehicleData, error) {
	// ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	// defer cancel()

	// b, err := v.identity.Account().Get(ctx, fmt.Sprintf("api/1/vehicles/%d/vehicle_data", id))
	// if err != nil {
	// 	return nil, err
	// }

	// var res tesla.VehicleData
	// if err := json.Unmarshal(b, &res); err != nil {
	// 	return nil, err
	// }

	var res tesla.VehicleData
	err := v.GetJSON(fmt.Sprintf("%sapi/1/vehicles/%d/vehicle_data", FleetAudienceEU, id), &res)

	return &res, err
}

func (v *API) commandPath(command string, id int64) string {
	return FleetAudienceEU + fmt.Sprintf("api/1/vehicles/%d/command/%s", id, command)
}

func (v *API) sendCommand(uri string) error {
	req, _ := request.New(http.MethodPost, uri, nil)

	var resp CommandResponse
	if err := v.DoJSON(req, &resp); err != nil {
		return err
	}

	if !resp.Response.Result && resp.Response.Reason != "" {
		return errors.New(resp.Response.Reason)
	}

	return nil
}

// StartCharging starts the charging of the vehicle after you have inserted the charging cable.
func (v *API) StartCharging(id int64) error {
	uri := v.commandPath("charge_start", id)
	return v.sendCommand(uri)
}

// StopCharging stops the charging of the vehicle.
func (v *API) StopCharging(id int64) error {
	uri := v.commandPath("charge_stop", id)
	return v.sendCommand(uri)
}

func (v *API) WakeUp(id int64) error {
	uri := FleetAudienceEU + fmt.Sprintf("api/1/vehicles/%d/wake_up", id)
	return v.sendCommand(uri)
}
