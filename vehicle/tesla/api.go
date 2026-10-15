package tesla

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util/request"
)

const FleetBaseUrl = "https://fleet-api.prd.na.vn.cloud.tesla.com"

// vehicle_data omits latitude/longitude unless location_data is requested (firmware 2023.38+)
const vehicleDataEndpoints = "charge_state;climate_state;drive_state;location_data;vehicle_state"

// API is a minimal Tesla Fleet API client, also used against a command proxy
type API struct {
	*request.Helper
	base string
}

// NewAPI creates a Fleet API client; the http client carries authentication
func NewAPI(client *http.Client, base string) *API {
	return &API{
		Helper: &request.Helper{Client: client},
		base:   strings.TrimSuffix(base, "/") + "/api/1",
	}
}

func (v *API) post(uri string, data any) error {
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	var res CommandResponse
	err = v.DoJSON(req, &res)

	// error reason is also delivered with non-200 status
	if reason := res.Reason(); reason != nil {
		if err != nil {
			return fmt.Errorf("%w: %w", err, reason)
		}
		return reason
	}

	return err
}

// Region returns the Fleet API region of the account
func (v *API) Region() (Region, error) {
	var res struct{ Response Region }
	err := v.GetJSON(v.base+"/users/region", &res)
	return res.Response, err
}

// Vehicles returns the account's vehicles
func (v *API) Vehicles() ([]Vehicle, error) {
	var res struct{ Response []Vehicle }
	err := v.GetJSON(v.base+"/vehicles", &res)
	return res.Response, err
}

// VehicleData returns the vehicle state including location
func (v *API) VehicleData(vin string) (*VehicleData, error) {
	var res VehicleData
	uri := fmt.Sprintf("%s/vehicles/%s/vehicle_data?endpoints=%s", v.base, vin, url.QueryEscape(vehicleDataEndpoints))
	if err := v.GetJSON(uri, &res); err != nil {
		return nil, err
	}
	if res.Error != "" {
		if res.ErrorDescription != "" {
			return nil, fmt.Errorf("%s: %s", res.Error, res.ErrorDescription)
		}
		return nil, errors.New(res.Error)
	}
	return &res, nil
}

// WakeUp wakes the vehicle
func (v *API) WakeUp(vin string) error {
	return v.post(fmt.Sprintf("%s/vehicles/%s/wake_up", v.base, vin), nil)
}

// Command sends a vehicle command
func (v *API) Command(vin, command string, data any) error {
	return v.post(fmt.Sprintf("%s/vehicles/%s/command/%s", v.base, vin, command), data)
}

// Products returns the account's products
func (v *API) Products() ([]Product, error) {
	var res struct{ Response []Product }
	err := v.GetJSON(v.base+"/products", &res)
	return res.Response, err
}

// EnergySiteStatus returns the energy site's live status
func (v *API) EnergySiteStatus(siteId int64) (EnergySiteStatus, error) {
	var res struct{ Response EnergySiteStatus }
	err := v.GetJSON(fmt.Sprintf("%s/energy_sites/%d/site_status", v.base, siteId), &res)
	return res.Response, err
}

// SetBatteryReserve sets the energy site's backup reserve
func (v *API) SetBatteryReserve(siteId int64, percent uint64) error {
	return v.post(fmt.Sprintf("%s/energy_sites/%d/backup", v.base, siteId), map[string]uint64{"backup_reserve_percent": percent})
}
