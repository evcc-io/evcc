package bluelink

import (
	"errors"
	"fmt"
	"net/http"
	"time"

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

func (v *API) Status(vid string) (StatusResponse, error) {
	var resp StatusResponse

	req, err := v.identity.Request(http.MethodGet, fmt.Sprintf(StatusURL, vid))
	if err == nil {
		if err = v.DoJSON(req, &resp); err == nil && resp.RetCode != resOK {
			err = fmt.Errorf("unexpected response: %s", resp.RetCode)
		}
	}

	return resp, err
}
