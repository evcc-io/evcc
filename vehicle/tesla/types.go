package tesla

import (
	tesla "github.com/evcc-io/tesla-proxy-client"
)

type (
	Vehicle         = tesla.Vehicle
	VehicleData     = tesla.VehicleData
	CommandResponse = tesla.CommandResponse
)

type RegionResponse struct {
	Response Region
}

type Region struct {
	Region          string
	FleetApiBaseUrl string `json:"fleet_api_base_url"`
}
