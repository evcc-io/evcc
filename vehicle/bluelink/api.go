package bluelink

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const (
	VehiclesURL     = "/api/v1/spa/vehicles"
	StatusURL       = "/api/v1/spa/vehicles/%s/status"
	StatusLatestURL = "/api/v1/spa/vehicles/%s/status/latest"
)

const (
	resOK      = "S"                    // auth fail: F
	timeFormat = "20060102150405 -0700" // Note: must add timeOffset
	timeOffset = " +0100"
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
	VIN, VehicleName, VehicleID string
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
	RetCode string
	ResCode string
	ResMsg  StatusData
}

type StatusLatestResponse struct {
	RetCode string
	ResCode string
	ResMsg  struct {
		VehicleStatusInfo struct {
			VehicleStatus StatusData
		}
	}
}

type StatusData struct {
	Time     string
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

func (d *StatusData) Updated() (time.Time, error) {
	return time.Parse(timeFormat, d.Time+timeOffset)
}

type DrivingDistance struct {
	RangeByFuel struct {
		EvModeRange struct {
			Value int
		}
	}
}

func (v *API) Status(vid string) (StatusLatestResponse, error) {
	var res StatusLatestResponse

	req, err := v.identity.Request(http.MethodGet, fmt.Sprintf(StatusLatestURL, vid))
	if err == nil {
		if err = v.DoJSON(req, &res); err == nil && res.RetCode != resOK {
			err = fmt.Errorf("unexpected response: %s", res.RetCode)
		}
	}

	return res, err
}

// StatusPartial refreshes the status from the bluelink api
func (v *API) StatusPartial(vid string) (StatusResponse, error) {
	var res StatusResponse

	req, err := v.identity.Request(http.MethodGet, fmt.Sprintf(StatusURL, vid))
	if err == nil {
		if err = v.DoJSON(req, &res); err == nil && res.RetCode != resOK {
			err = fmt.Errorf("unexpected response: %s", res.RetCode)
		}
	}

	return res, err
}
