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
	VehiclesURL         = "vehicles"
	StatusURL           = "vehicles/%s/status"
	StatusLatestURL     = "vehicles/%s/status/latest"
	StatusURLCCS2       = "vehicles/%s/ccs2/carstatus"
	StatusLatestURLCCS2 = "vehicles/%s/ccs2/carstatus/latest"
)

const (
	resOK = "S" // auth fail: F
)

// ErrAuthFail indicates authorization failure
var ErrAuthFail = errors.New("authorization failed")

// API implements the Kia/Hyundai bluelink api.
type API struct {
	*request.Helper
	baseURI string
}

// New creates a new BlueLink API
func NewAPI(log *util.Logger, baseURI string, decorator func(*http.Request) error) *API {
	v := &API{
		Helper:  request.NewHelper(log),
		baseURI: strings.TrimSuffix(baseURI, "/api/v1/spa") + "/api/v1/spa",
	}

	// api is unbelievably slow when retrieving status
	v.Client.Timeout = 120 * time.Second

	v.Client.Transport = &transport.Decorator{
		Decorator: decorator,
		Base:      v.Client.Transport,
	}

	return v
}

type Vehicle struct {
	VIN, VehicleName, VehicleID string
	CcuCCS2ProtocolSupport      int
}

func (v *API) Vehicles() ([]Vehicle, error) {
	var res VehiclesResponse

	uri := fmt.Sprintf("%s/%s", v.baseURI, VehiclesURL)
	err := v.GetJSON(uri, &res)

	return res.ResMsg.Vehicles, err
}

// StatusLatest retrieves the latest server-side status
func (v *API) StatusLatest(vehicle Vehicle) (BluelinkVehicleStatusLatest, error) {
	vid := vehicle.VehicleID

	if vehicle.CcuCCS2ProtocolSupport != 0 {
		var res StatusLatestResponseCCS
		uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusLatestURLCCS2, vid))
		err := v.GetJSON(uri, &res)
		if err == nil && res.RetCode != resOK {
			err = fmt.Errorf("unexpected response: %s", res.RetCode)
		}
		return res, err
	}

	var res StatusLatestResponse
	uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusLatestURL, vid))
	err := v.GetJSON(uri, &res)
	if err == nil && res.RetCode != resOK {
		err = fmt.Errorf("unexpected response: %s", res.RetCode)
	}
	return res, err
}

// StatusPartial refreshes the status
func (v *API) StatusPartial(vehicle Vehicle) (BluelinkVehicleStatus, error) {
	vid := vehicle.VehicleID

	if vehicle.CcuCCS2ProtocolSupport != 0 {
		var res StatusLatestResponseCCS
		uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusLatestURLCCS2, vid))
		err := v.GetJSON(uri, &res)
		if err == nil && res.RetCode != resOK {
			err = fmt.Errorf("unexpected response: %s", res.RetCode)
		}
		return res, err
	}

	var res StatusResponse
	uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusURL, vid))
	err := v.GetJSON(uri, &res)
	if err == nil && res.RetCode != resOK {
		err = fmt.Errorf("unexpected response: %s", res.RetCode)
	}
	return res.ResMsg, err
}
