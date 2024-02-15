package vc

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	tesla "github.com/evcc-io/tesla-proxy-client"
	"golang.org/x/oauth2"
)

const (
	FleetAudienceEU = "https://fleet-api.prd.eu.vn.cloud.tesla.com"
)

type API struct {
	*request.Helper
	base string
}

func NewAPI(log *util.Logger, ts oauth2.TokenSource, timeout time.Duration) *API {
	client := request.NewHelper(log)
	client.Client.Timeout = timeout
	client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   client.Transport,
	}

	return &API{
		Helper: client,
		base:   FleetAudienceEU,
	}
}

func (v *API) Region() (Region, error) {
	var res RegionResponse
	err := v.GetJSON(fmt.Sprintf("%s/api/1/users/region", FleetAudienceEU), &res)
	if err == nil {
		v.base = res.Response.FleetApiBaseUrl
	}
	return res.Response, err
}

func (v *API) Vehicles() ([]*Vehicle, error) {
	var res tesla.VehiclesResponse
	err := v.GetJSON(fmt.Sprintf("%s/api/1/vehicles", v.base), &res)

	return res.Response, err
}

func (v *API) VehicleData(id int64) (*VehicleData, error) {
	var res tesla.VehicleData
	err := v.GetJSON(fmt.Sprintf("%s/api/1/vehicles/%d/vehicle_data", v.base, id), &res)

	return &res, err
}

func (v *API) WakeUp(id int64) (*VehicleData, error) {
	var res tesla.VehicleData
	err := v.GetJSON(fmt.Sprintf("%s/api/1/vehicles/%d/wake_up", v.base, id), &res)

	return &res, err
}
