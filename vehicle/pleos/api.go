package pleos

import (
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

// BaseURL is the Pleos Vehicle Data API base URL (Hyundai/Kia/Genesis, EU Data Act)
// https://document.pleos.ai/en/api-reference/vehicle-data-api/intro
const BaseURL = "https://api.pleos.ai/v1"

// API is the Pleos Vehicle Data API client
type API struct {
	*request.Helper
}

// NewAPI creates a Pleos API client. The brand (hyundai, kia, genesis) is sent
// as the Brand header required by all endpoints.
func NewAPI(log *util.Logger, ts oauth2.TokenSource, brand string) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Transport = &oauth2.Transport{
		Source: ts,
		Base: &transport.Decorator{
			Decorator: transport.DecorateHeaders(map[string]string{
				"Brand": brand,
			}),
			Base: v.Transport,
		},
	}

	return v
}

// get decodes the data envelope of an endpoint
func get[T any](v *API, path string) (T, error) {
	var res struct {
		Data T `json:"data"`
	}
	err := v.GetJSON(BaseURL+"/vehicles/"+path, &res)
	return res.Data, err
}

// Vehicles returns the vehicles the key has consented access to
func (v *API) Vehicles() ([]Vehicle, error) {
	res, err := get[struct {
		Vehicles []Vehicle `json:"vehicles"`
	}](v, "consent")
	return res.Vehicles, err
}

// Battery returns the battery charging status
func (v *API) Battery(vin string) (Battery, error) {
	return get[Battery](v, vin+"/batteries")
}

// Powertrain returns the powertrain status including range
func (v *API) Powertrain(vin string) (Powertrain, error) {
	return get[Powertrain](v, vin+"/powertrains")
}

// Driving returns the vehicle operation status including odometer
func (v *API) Driving(vin string) (Driving, error) {
	return get[Driving](v, vin+"/driving")
}

// Location returns the vehicle location
func (v *API) Location(vin string) (Location, error) {
	return get[Location](v, vin+"/locations")
}

// Status returns the sensor-based vehicle status including climate control
func (v *API) Status(vin string) (Status, error) {
	return get[Status](v, vin+"/status")
}
