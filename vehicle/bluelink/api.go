package bluelink

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

const (
	VehiclesURL     = "vehicles"
	StatusURL       = "vehicles/%s/status"
	StatusLatestURL = "vehicles/%s/status/latest"
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
	baseURI string
}

type Requester interface {
	Request(*http.Request) error
}

// New creates a new BlueLink API
func NewAPI(log *util.Logger, baseURI string, identity Requester, cache time.Duration) *API {
	v := &API{
		Helper:  request.NewHelper(log),
		baseURI: strings.TrimRight(baseURI, "/api/v1/spa") + "/api/v1/spa",
	}

	// api is unbelievably slow when retrieving status
	v.Client.Timeout = 120 * time.Second

	v.Client.Transport = &transport.Decorator{
		Decorator: identity.Request,
		Base:      v.Client.Transport,
	}

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
	var res VehiclesResponse

	uri := fmt.Sprintf("%s/%s", v.baseURI, VehiclesURL)
	err := v.GetJSON(uri, &res)

	return res.ResMsg.Vehicles, err
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

	uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusLatestURL, vid))
	err := v.GetJSON(uri, &res)
	if err == nil && res.RetCode != resOK {
		err = fmt.Errorf("unexpected response: %s", res.RetCode)
	}

	return res, err
}

// StatusPartial refreshes the status from the bluelink api
func (v *API) StatusPartial(vid string) (StatusResponse, error) {
	var res StatusResponse

	uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusURL, vid))
	err := v.GetJSON(uri, &res)
	if err == nil && res.RetCode != resOK {
		err = fmt.Errorf("unexpected response: %s", res.RetCode)
	}

	return res, err
}
