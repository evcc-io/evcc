package vc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bogosj/tesla"
	"github.com/evcc-io/evcc/util/request"
	"github.com/teslamotors/vehicle-command/pkg/account"
)

type API struct {
	user *account.Account
}

func NewAPI(user *account.Account) *API {
	return &API{
		user: user,
	}
}

func (v *API) Vehicles() ([]*tesla.Vehicle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	b, err := v.user.Get(ctx, "api/1/vehicles")
	if err != nil {
		return nil, err
	}

	var res tesla.VehiclesResponse
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}

	return res.Response, nil
}

func (v *API) VehicleData(id int64) (*tesla.VehicleData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	b, err := v.user.Get(ctx, fmt.Sprintf("api/1/vehicles/%d/vehicle_data", id))
	if err != nil {
		return nil, err
	}

	var res tesla.VehicleData
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
