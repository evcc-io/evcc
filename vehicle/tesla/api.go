package tesla

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const FleetBaseUrl = "https://fleet-api.prd.na.vn.cloud.tesla.com"

const vehicleDataEndpoints = "charge_state;climate_state;drive_state;vehicle_state"

// API is a minimal Tesla Fleet API client, also used against a command proxy
type API struct {
	*request.Helper
	log  *util.Logger
	base string

	noLocation atomic.Bool // app lacks the vehicle location scope
}

// NewAPI creates a Fleet API client; the http client carries authentication
func NewAPI(log *util.Logger, client *http.Client, base string) *API {
	return &API{
		Helper: &request.Helper{Client: client},
		log:    log,
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

// VehicleData returns the vehicle state including location if the app has the location scope
func (v *API) VehicleData(vin string) (*VehicleData, error) {
	endpoints := vehicleDataEndpoints

	// vehicle_data omits latitude/longitude unless location_data is requested (firmware 2023.38+)
	location := !v.noLocation.Load()
	if location {
		endpoints += ";location_data"
	}

	var res VehicleData
	uri := fmt.Sprintf("%s/vehicles/%s/vehicle_data?endpoints=%s", v.base, vin, url.QueryEscape(endpoints))
	if err := v.GetJSON(uri, &res); err != nil {
		// Tesla rejects the whole request with 403 if the location scope is missing
		var se *request.StatusError
		if location && errors.As(err, &se) && se.HasStatus(http.StatusForbidden) {
			v.log.WARN.Println("vehicle location unavailable: Fleet API app lacks the vehicle_location scope")
			v.noLocation.Store(true)
			return v.VehicleData(vin)
		}
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

// EnergySiteInfo returns the energy site's configuration
func (v *API) EnergySiteInfo(siteId int64) (EnergySiteInfo, error) {
	var res struct{ Response EnergySiteInfo }
	err := v.GetJSON(fmt.Sprintf("%s/energy_sites/%d/site_info", v.base, siteId), &res)
	return res.Response, err
}

// SetBatteryReserve sets the energy site's backup reserve
func (v *API) SetBatteryReserve(siteId int64, percent uint64) error {
	return v.post(fmt.Sprintf("%s/energy_sites/%d/backup", v.base, siteId), map[string]uint64{"backup_reserve_percent": percent})
}
