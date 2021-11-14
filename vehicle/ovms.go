package vehicle

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/net/publicsuffix"
)

type ovmsStatusResponse struct {
	Odometer string `json:"odometer"`
}

type ovmsChargeResponse struct {
	ChargeEtrFull    string `json:"charge_etr_full"`
	ChargeState      string `json:"chargestate"`
	ChargePortOpen   int    `json:"cp_dooropen"`
	EstimatedRange   string `json:"estimatedrange"`
	MessageAgeServer int    `json:"m_msgage_s"`
	Soc              string `json:"soc"`
}

type ovmsConnectResponse struct {
	NetConnected int `json:"v_net_connected"`
}

// OVMS is an api.Vehicle implementation for dexters-web server requests
type Ovms struct {
	*embed
	*request.Helper
	user, password, vehicleId, server string
	cache                             time.Duration
	isOnline                          bool
	chargeG                           func() (interface{}, error)
	statusG                           func() (interface{}, error)
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

	v.chargeG = provider.NewCached(v.batteryAPI, cc.Cache).InterfaceGetter()
	v.statusG = provider.NewCached(v.statusAPI, cc.Cache).InterfaceGetter()

	var err error
	v.Jar, err = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	return v, err
}

func (v *Ovms) loginToServer() (err error) {
	uri := fmt.Sprintf("https://%s:6869/api/cookie?username=%s&password=%s", v.server, v.user, v.password)

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
func (v *Ovms) batteryAPI() (interface{}, error) {
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
func (v *Ovms) statusAPI() (interface{}, error) {
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

// SoC implements the api.Vehicle interface
func (v *Ovms) SoC() (float64, error) {
	res, err := v.chargeG()

	if res, ok := res.(ovmsChargeResponse); err == nil && ok {
		return strconv.ParseFloat(res.Soc, 64)
	}

	return 0, err
}

var _ api.ChargeState = (*Ovms)(nil)

// Status implements the api.ChargeState interface
func (v *Ovms) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.chargeG()
	if res, ok := res.(ovmsChargeResponse); err == nil && ok {
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

	if res, ok := res.(ovmsChargeResponse); err == nil && ok {
		return strconv.ParseInt(res.EstimatedRange, 0, 64)
	}

	return 0, err
}

var _ api.VehicleOdometer = (*Ovms)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Ovms) Odometer() (float64, error) {
	res, err := v.statusG()

	if res, ok := res.(ovmsStatusResponse); err == nil && ok {
		odometer, err := strconv.ParseFloat(res.Odometer, 64)
		if err == nil {
			return odometer / 10, nil
		}
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Ovms)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Ovms) FinishTime() (time.Time, error) {
	res, err := v.chargeG()

	if res, ok := res.(ovmsChargeResponse); err == nil && ok {
		cef, err := strconv.ParseInt(res.ChargeEtrFull, 0, 64)
		if err == nil {
			return time.Now().Add(time.Duration(cef) * time.Minute), nil
		}
	}

	return time.Time{}, err
}
