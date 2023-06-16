package vehicle

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/net/publicsuffix"
)

type ovmsStatusResponse struct {
	Odometer float64 `json:"odometer,string"`
}

type ovmsChargeResponse struct {
	ChargeEtrFull    int64   `json:"charge_etr_full,string"`
	ChargeState      string  `json:"chargestate"`
	ChargePortOpen   int     `json:"cp_dooropen"`
	EstimatedRange   string  `json:"estimatedrange"`
	MessageAgeServer int     `json:"m_msgage_s"`
	Soc              float64 `json:"soc,string"`
}

type ovmsLocationResponse struct {
	Latitude  float64 `json:"latitude,string"`
	Longitude float64 `json:"longitude,string"`
}

type ovmsConnectResponse struct {
	NetConnected int `json:"v_net_connected"`
}

// OVMS is an api.Vehicle implementation for dexters-web server requests
type Ovms struct {
	*embed
	*request.Helper
	user, password    string
	vehicleId, server string
	cache             time.Duration
	isOnline          bool
	chargeG           func() (ovmsChargeResponse, error)
	statusG           func() (ovmsStatusResponse, error)
	locationG         func() (ovmsLocationResponse, error)
}

func init() {
	registry.Add("ovms", NewOvmsFromConfig)
}

// NewOVMSFromConfig creates a new vehicle
func NewOvmsFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed                             `mapstructure:",squash"`
		User, Password, VehicleID, Server string
		Cache                             time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("ovms").Redact(cc.User, cc.Password, cc.VehicleID)

	v := &Ovms{
		embed:     &cc.embed,
		Helper:    request.NewHelper(log),
		user:      cc.User,
		password:  cc.Password,
		vehicleId: cc.VehicleID,
		server:    cc.Server,
		cache:     cc.Cache,
	}

	v.chargeG = provider.Cached(v.batteryAPI, cc.Cache)
	v.statusG = provider.Cached(v.statusAPI, cc.Cache)
	v.locationG = provider.Cached(v.locationAPI, cc.Cache)

	var err error
	v.Jar, err = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	return v, err
}

func (v *Ovms) loginToServer() (err error) {
	uri := fmt.Sprintf("https://%s:6869/api/cookie?username=%s&password=%s", v.server, url.QueryEscape(v.user), url.QueryEscape(v.password))

	var resp *http.Response
	if resp, err = v.Get(uri); err == nil {
		resp.Body.Close()
	}

	return err
}

func (v *Ovms) connectRequest() (ovmsConnectResponse, error) {
	uri := fmt.Sprintf("https://%s:6869/api/vehicle/%s", v.server, v.vehicleId)
	var res ovmsConnectResponse
	err := v.GetJSON(uri, &res)
	return res, err
}

func (v *Ovms) chargeRequest() (ovmsChargeResponse, error) {
	uri := fmt.Sprintf("https://%s:6869/api/charge/%s", v.server, v.vehicleId)
	var res ovmsChargeResponse
	err := v.GetJSON(uri, &res)
	return res, err
}

func (v *Ovms) statusRequest() (ovmsStatusResponse, error) {
	uri := fmt.Sprintf("https://%s:6869/api/status/%s", v.server, v.vehicleId)
	var res ovmsStatusResponse
	err := v.GetJSON(uri, &res)
	return res, err
}

func (v *Ovms) locationRequest() (ovmsLocationResponse, error) {
	uri := fmt.Sprintf("https://%s:6869/api/location/%s", v.server, v.vehicleId)
	var res ovmsLocationResponse
	err := v.GetJSON(uri, &res)
	return res, err
}

func (v *Ovms) authFlow() error {
	var resp ovmsConnectResponse
	err := v.loginToServer()
	if err == nil {
		resp, err = v.connectRequest()
		if err == nil {
			v.isOnline = resp.NetConnected == 1
		}
	}
	return err
}

// batteryAPI provides battery-status api response
func (v *Ovms) batteryAPI() (ovmsChargeResponse, error) {
	var resp ovmsChargeResponse

	resp, err := v.chargeRequest()
	if err != nil {
		err = v.authFlow()
		if err == nil {
			resp, err = v.chargeRequest()
		}
	}

	messageAge := time.Duration(resp.MessageAgeServer) * time.Second
	if err == nil && v.isOnline && messageAge > v.cache {
		err = api.ErrMustRetry
	}

	return resp, err
}

// statusAPI provides vehicle status api response
func (v *Ovms) statusAPI() (ovmsStatusResponse, error) {
	var resp ovmsStatusResponse

	resp, err := v.statusRequest()
	if err != nil {
		err = v.authFlow()
		if err == nil {
			resp, err = v.statusRequest()
		}
	}

	return resp, err
}

// location API provides vehicle position api response
func (v *Ovms) locationAPI() (ovmsLocationResponse, error) {
	var resp ovmsLocationResponse

	resp, err := v.locationRequest()
	if err != nil {
		err = v.authFlow()
		if err == nil {
			resp, err = v.locationRequest()
		}
	}

	return resp, err
}

// Soc implements the api.Vehicle interface
func (v *Ovms) Soc() (float64, error) {
	res, err := v.chargeG()
	return res.Soc, err
}

var _ api.ChargeState = (*Ovms)(nil)

// Status implements the api.ChargeState interface
func (v *Ovms) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.chargeG()
	if err == nil {
		if res.ChargePortOpen > 0 {
			status = api.StatusB
		}
		if res.ChargeState == "charging" {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.VehicleRange = (*Ovms)(nil)

// Range implements the api.VehicleRange interface
func (v *Ovms) Range() (int64, error) {
	res, err := v.chargeG()

	if err == nil {
		return strconv.ParseInt(res.EstimatedRange, 0, 64)
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Ovms)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Ovms) Odometer() (float64, error) {
	res, err := v.statusG()
	return res.Odometer / 10, err
}

var _ api.VehicleFinishTimer = (*Ovms)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Ovms) FinishTime() (time.Time, error) {
	res, err := v.chargeG()
	return time.Now().Add(time.Duration(res.ChargeEtrFull) * time.Minute), err
}

// VehiclePosition returns the vehicles position in latitude and longitude
func (v *Ovms) Position() (float64, float64, error) {
	res, err := v.locationG()
	return res.Latitude, res.Longitude, err
}
