package vc

import (
	"fmt"
	"time"

	"github.com/bogosj/tesla"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	FleetAudienceEU = "https://fleet-api.prd.eu.vn.cloud.tesla.com"
)

type API struct {
	*request.Helper
	identity *Identity
}

func NewAPI(log *util.Logger, identity *Identity, timeout time.Duration) *API {
	client := request.NewHelper(log)
	client.Client.Timeout = timeout
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
	// ctx, cancel := context.WithTimeout(context.Background(), v.Timeout)
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
	err := v.GetJSON(fmt.Sprintf("%s/api/1/vehicles", FleetAudienceEU), &res)

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
	err := v.GetJSON(fmt.Sprintf("%s/api/1/vehicles/%d/vehicle_data", FleetAudienceEU, id), &res)

	return &res, err
}

func (v *API) WakeUp(id int64) (*VehicleData, error) {
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
	err := v.GetJSON(fmt.Sprintf("%s/api/1/vehicles/%d/wake_up", FleetAudienceEU, id), &res)

	return &res, err
}
