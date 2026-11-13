package bluelink

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

const (
	VehiclesURL         = "vehicles"
	StatusURL           = "vehicles/%s/status"                // Triggers refresh from vehicle (older API)
	StatusLatestURL     = "vehicles/%s/status/latest"         // Cached data with location/odometer (older API)
	StatusURLCCS2       = "vehicles/%s/ccs2/carstatus"        // Triggers refresh from vehicle (newer API)
	StatusLatestURLCCS2 = "vehicles/%s/ccs2/carstatus/latest" // Newer API (2024+ vehicles with ccOS)
)

const (
	resOK          = "S"    // auth fail: F
	resUnavailable = "5031" // service temporary unavailable
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

	if transport, ok := v.Client.Transport.(*http.Transport); ok {
		transport.TLSHandshakeTimeout = 30 * time.Second
	}

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

// StatusLatest retrieves vehicle status (triggers refresh for older API, then returns cached data)
func (v *API) StatusLatest(vehicle Vehicle) (BluelinkVehicleStatusLatest, error) {
	if vehicle.CcuCCS2ProtocolSupport != 0 {
		return v.statusLatestCCS2(vehicle.VehicleID)
	}

	// For older API: first trigger refresh, then get latest cached data
	_ = v.Refresh(vehicle) // Ignore error, will retry with /status/latest

	var res StatusLatestResponse
	uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusLatestURL, vehicle.VehicleID))
	err := v.GetJSON(uri, &res)

	// some flag-0 cars are only served by the CCS2 endpoint, see Hyundai-Kia-Connect/hyundai_kia_connect_api#1546
	if err != nil && res.ResCode == resUnavailable {
		return v.statusLatestCCS2(vehicle.VehicleID)
	}

	if err == nil && res.RetCode != resOK {
		err = fmt.Errorf("unexpected response: %s", res.RetCode)
	}
	return res, err
}

func (v *API) statusLatestCCS2(vid string) (BluelinkVehicleStatusLatest, error) {
	var res StatusLatestResponseCCS
	uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusLatestURLCCS2, vid))
	err := v.GetJSON(uri, &res)
	if err == nil && res.RetCode != resOK {
		err = fmt.Errorf("unexpected response: %s", res.RetCode)
	}
	return res, err
}

// Refresh triggers a status update from the vehicle
func (v *API) Refresh(vehicle Vehicle) error {
	path := StatusURL
	if vehicle.CcuCCS2ProtocolSupport != 0 {
		// CCS2 cars wake via the CCS2 endpoint, see Hyundai-Kia-Connect/hyundai_kia_connect_api@cadd705
		path = StatusURLCCS2
	}

	var res StatusResponse
	uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(path, vehicle.VehicleID))
	err := v.GetJSON(uri, &res)
	if err == nil && res.RetCode != resOK {
		err = fmt.Errorf("unexpected response: %s", res.RetCode)
	}
	return err
}

// ChargeEnable starts or stops charging
func (v *API) ChargeEnable(identity *Identity, vehicle Vehicle, pin string, enable bool) error {
	action := "stop"
	if enable {
		action = "start"
	}

	uri := fmt.Sprintf("%s/vehicles/%s/control/charge", v.baseURI, vehicle.VehicleID)
	data := map[string]string{"action": action, "deviceId": identity.deviceID}

	ccs2 := vehicle.CcuCCS2ProtocolSupport != 0
	if ccs2 {
		uri = fmt.Sprintf("%s/vehicles/%s/ccs2/control/charge", strings.Replace(v.baseURI, "/api/v1/", "/api/v2/", 1), vehicle.VehicleID)
		data = map[string]string{"command": action}
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	if ccs2 {
		token, err := identity.controlToken(pin)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", token)
		req.Header.Set("AuthorizationCCSP", token)
		req.Header.Set("Ccuccs2protocolsupport", strconv.Itoa(vehicle.CcuCCS2ProtocolSupport))
	}

	var res struct{ RetCode string }
	if err := v.DoJSON(req, &res); err != nil {
		return err
	}
	if res.RetCode != resOK {
		return fmt.Errorf("unexpected response: %s", res.RetCode)
	}

	return nil
}
