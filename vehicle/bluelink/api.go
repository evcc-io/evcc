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
	resOK = "S" // auth fail: F
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
func NewAPI(log *util.Logger, baseURI string, identity Requester) *API {
	v := &API{
		Helper:  request.NewHelper(log),
		baseURI: strings.TrimSuffix(baseURI, "/api/v1/spa") + "/api/v1/spa",
	}

	// api is unbelievably slow when retrieving status
	v.Client.Timeout = 120 * time.Second

	v.Client.Transport = &transport.Decorator{
		Decorator: identity.Request,
		Base:      v.Client.Transport,
	}

	return v
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

// StatusLatest retrieves the latest server-side status
func (v *API) StatusLatest(vid string) (StatusLatestResponse, error) {
	var res StatusLatestResponse

	uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusLatestURL, vid))
	err := v.GetJSON(uri, &res)
	if err == nil && res.RetCode != resOK {
		err = fmt.Errorf("unexpected response: %s", res.RetCode)
	}

	return res, err
}

// StatusPartial refreshes the status
func (v *API) StatusPartial(vid string) (StatusResponse, error) {
	var res StatusResponse

	uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusURL, vid))
	err := v.GetJSON(uri, &res)
	if err == nil && res.RetCode != resOK {
		err = fmt.Errorf("unexpected response: %s", res.RetCode)
	}

	return res, err
}
