package bluelink

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

const (
	VehiclesURL         = "vehicles"
	StatusURL           = "vehicles/%s/status"                // Triggers refresh from vehicle (older API)
	StatusLatestURL     = "vehicles/%s/status/latest"         // Cached data with location/odometer (older API)
	StatusLatestURLCCS2 = "vehicles/%s/ccs2/carstatus/latest" // Newer API (2024+ vehicles with ccOS)
)

const (
	resOK = "S" // auth fail: F
)

// ErrAuthFail indicates authorization failure
var ErrAuthFail = errors.New("authorization failed")

// serialCharge sends at most one charge command at a time and replaces queued
// commands with the latest requested start/stop so an older request cannot apply last.
type serialCharge struct {
	mu      sync.Mutex
	cond    *sync.Cond
	busy    bool
	pending *bool
	err     error
}

func newSerialCharge() *serialCharge {
	s := &serialCharge{}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *serialCharge) Do(enable bool, fn func(bool) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := enable
	s.pending = &e

	if s.busy {
		for s.busy {
			s.cond.Wait()
		}
		return s.err
	}

	for s.pending != nil {
		want := *s.pending
		s.pending = nil
		s.busy = true
		err := func() error {
			s.mu.Unlock()
			defer s.mu.Lock()
			return fn(want)
		}()
		s.err = err
	}

	s.busy = false
	s.cond.Broadcast()
	return s.err
}

// API implements the Kia/Hyundai bluelink api.
type API struct {
	*request.Helper
	baseURI  string
	mu       sync.Mutex
	chargeMu map[string]*serialCharge
}

// New creates a new BlueLink API
func NewAPI(log *util.Logger, baseURI string, decorator func(*http.Request) error) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		baseURI:  strings.TrimSuffix(baseURI, "/api/v1/spa") + "/api/v1/spa",
		chargeMu: make(map[string]*serialCharge),
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

	// For older API: first trigger refresh, then get latest cached data
	_ = v.Refresh(vehicle) // Ignore error, will retry with /status/latest

	var res StatusLatestResponse
	uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusLatestURL, vid))
	err := v.GetJSON(uri, &res)
	if err == nil && res.RetCode != resOK {
		err = fmt.Errorf("unexpected response: %s", res.RetCode)
	}
	return res, err
}

// Refresh triggers a status update from the vehicle
func (v *API) Refresh(vehicle Vehicle) error {
	var res StatusResponse
	uri := fmt.Sprintf("%s/%s", v.baseURI, fmt.Sprintf(StatusURL, vehicle.VehicleID))
	err := v.GetJSON(uri, &res)
	if err == nil && res.RetCode != resOK {
		err = fmt.Errorf("unexpected response: %s", res.RetCode)
	}
	return err
}

type chargeControlResponse struct {
	RetCode string `json:"retCode"`
	MsgID   string `json:"msgId"`
}

func spaV2URI(baseURI string) string {
	return strings.Replace(baseURI, "/api/v1/spa", "/api/v2/spa", 1)
}

func chargeControlURI(baseURI, vehicleID string, ccs2 bool) string {
	if ccs2 {
		return fmt.Sprintf("%s/vehicles/%s/ccs2/control/charge", spaV2URI(baseURI), vehicleID)
	}
	return fmt.Sprintf("%s/vehicles/%s/control/charge", baseURI, vehicleID)
}

func (v *API) serial(id string) *serialCharge {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.chargeMu == nil {
		v.chargeMu = make(map[string]*serialCharge)
	}
	s, ok := v.chargeMu[id]
	if !ok {
		s = newSerialCharge()
		v.chargeMu[id] = s
	}
	return s
}

func decodeChargeControl(body []byte) error {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}

	var res chargeControlResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return err
	}
	if res.RetCode == "" {
		return fmt.Errorf("missing retCode")
	}
	if res.RetCode != resOK {
		return fmt.Errorf("unexpected response: %s", res.RetCode)
	}
	return nil
}

// ChargeEnable starts or stops charging via CCS2 or the older Bluelink charge control API.
func (v *API) ChargeEnable(identity *Identity, vehicle Vehicle, pin string, enable bool) error {
	return v.serial(vehicle.VehicleID).Do(enable, func(enable bool) error {
		return v.sendChargeEnable(identity, vehicle, pin, enable)
	})
}

func (v *API) sendChargeEnable(identity *Identity, vehicle Vehicle, pin string, enable bool) error {
	if vehicle.CcuCCS2ProtocolSupport != 0 {
		err := v.postChargeControl(identity, vehicle, pin, enable, true)
		if err == nil {
			return nil
		}
		if pin == "" {
			return err
		}

		if err2 := v.postChargeControl(identity, vehicle, pin, enable, false); err2 != nil {
			return err2
		}
		return nil
	}

	return v.postChargeControl(identity, vehicle, pin, enable, false)
}

func (v *API) postChargeControl(identity *Identity, vehicle Vehicle, pin string, enable, ccs2 bool) error {
	ctx := context.Background()
	if ccs2 && pin != "" {
		token, err := identity.ControlAuthorization(pin)
		if err != nil {
			return err
		}
		ctx = WithControlToken(ctx, token)
	}

	var payload map[string]string
	if ccs2 {
		cmd := map[bool]string{true: "start", false: "stop"}
		payload = map[string]string{"command": cmd[enable]}
	} else {
		action := map[bool]string{true: "start", false: "stop"}
		payload = map[string]string{"action": action[enable]}
		if id := identity.DeviceID(); id != "" {
			payload["deviceId"] = id
		}
	}

	req, err := request.New(http.MethodPost, chargeControlURI(v.baseURI, vehicle.VehicleID, ccs2), request.MarshalJSON(payload), request.JSONEncoding)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	body, err := v.DoBody(req)
	if err != nil {
		return err
	}

	return decodeChargeControl(body)
}
