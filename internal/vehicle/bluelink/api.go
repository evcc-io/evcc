package bluelink

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

const resOK = "S" // auth fail: F

const (
	VehiclesURL = "/api/v1/spa/vehicles"
	StatusURL   = "/api/v1/spa/vehicles/%s/status"
)

// ErrAuthFail indicates authorization failure
var ErrAuthFail = errors.New("authorization failed")

// API implements the Kia/Hyundai bluelink api.
// Based on https://github.com/Hacksore/bluelinky.
type API struct {
	*request.Helper
	log      *util.Logger
	identity *Identity
	apiG     func() (interface{}, error)
	Vehicle  Vehicle
}

// New creates a new BlueLink API
func NewAPI(log *util.Logger, identity *Identity, cache time.Duration) *API {
	v := &API{
		log:      log,
		identity: identity,
		Helper:   request.NewHelper(log),
	}

	// api is unbelievably slow when retrieving status
	v.Helper.Client.Timeout = 120 * time.Second

	v.apiG = provider.NewCached(v.statusAPI, cache).InterfaceGetter()

	return v
}

type VehiclesResponse struct {
	RetCode string
	ResMsg  struct {
		Vehicles []Vehicle
	}
}

type Vehicle struct {
	Vin, VehicleName, VehicleID string
}

func (v *API) Vehicles() ([]Vehicle, error) {
	req, err := v.identity.Request(http.MethodGet, VehiclesURL)

	var resp VehiclesResponse
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	return resp.ResMsg.Vehicles, err
}

type StatusResponse struct {
	timestamp time.Time // add missing timestamp
	RetCode   string
	ResMsg    struct {
		EvStatus struct {
			BatteryStatus float64
			RemainTime2   struct {
				Atc struct {
					Value, Unit int
				}
			}
			DrvDistance []DrivingDistance
		}
		Vehicles []Vehicle
	}
}

type DrivingDistance struct {
	RangeByFuel struct {
		EvModeRange struct {
			Value int
		}
	}
}

func (v *API) getStatus() (StatusResponse, error) {
	var resp StatusResponse

	req, err := v.identity.Request(http.MethodGet, fmt.Sprintf(StatusURL, v.Vehicle.VehicleID))
	if err == nil {
		if err = v.DoJSON(req, &resp); err == nil && resp.RetCode != resOK {
			err = fmt.Errorf("unexpected response: %s", resp.RetCode)
		}
	}

	return resp, err
}

// status retrieves the bluelink status response
func (v *API) statusAPI() (interface{}, error) {
	res, err := v.getStatus()
	res.timestamp = time.Now() // add local timestamp to cache for FinishTime

	return res, err
}

var _ api.Battery = (*API)(nil)

// SoC implements the api.Vehicle interface
func (v *API) SoC() (float64, error) {
	res, err := v.apiG()

	if res, ok := res.(StatusResponse); err == nil && ok {
		return float64(res.ResMsg.EvStatus.BatteryStatus), nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*API)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *API) FinishTime() (time.Time, error) {
	res, err := v.apiG()

	if res, ok := res.(StatusResponse); err == nil && ok {
		remaining := res.ResMsg.EvStatus.RemainTime2.Atc.Value

		if remaining == 0 {
			return time.Time{}, api.ErrNotAvailable
		}

		return res.timestamp.Add(time.Duration(remaining) * time.Minute), nil
	}

	return time.Time{}, err
}

var _ api.VehicleRange = (*API)(nil)

// Range implements the api.VehicleRange interface
func (v *API) Range() (int64, error) {
	res, err := v.apiG()

	if res, ok := res.(StatusResponse); err == nil && ok {
		if dist := res.ResMsg.EvStatus.DrvDistance; len(dist) == 1 {
			return int64(dist[0].RangeByFuel.EvModeRange.Value), nil
		}

		return 0, api.ErrNotAvailable
	}

	return 0, err
}
