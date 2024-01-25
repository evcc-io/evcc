package vc

import "github.com/bogosj/tesla"

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
