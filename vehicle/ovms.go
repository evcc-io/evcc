package vehicle

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/net/publicsuffix"
)

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
	log                               *util.Logger
	user, password, vehicleId, server string
	chargeG                           func() (interface{}, error)
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

	log := util.NewLogger("ovms")

	v := &Ovms{
		embed:     &cc.embed,
		log:       log,
		user:      cc.User,
		password:  cc.Password,
		vehicleId: cc.VehicleID,
		server:    cc.Server,
	}

	v.chargeG = provider.NewCached(v.batteryAPI, cc.Cache).InterfaceGetter()

	return v, nil
}

func (v *Ovms) getCookies() (cookieClient *request.Helper, err error) {
	cookieClient = request.NewHelper(v.log)
	cookieClient.Client.Jar, err = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	if err == nil {
		uri := fmt.Sprintf("http://%s:6868/api/cookie?username=%s&password=%s", v.server, v.user, v.password)

		var resp *http.Response
		if resp, err = cookieClient.Get(uri); err == nil {
			resp.Body.Close()
		}
	}

	return cookieClient, err
}

func (v *Ovms) delete(url string, cookieClient *request.Helper) error {
	req, err := request.New(http.MethodDelete, url, nil)
	if err == nil {
		_, err = cookieClient.Do(req)
	}
	return err
}

func (v *Ovms) authFlow() (*request.Helper, bool, error) {
	cookieClient, err := v.getCookies()
	if err == nil {
		var resp ovmsConnectResponse
		resp, err = v.connectRequest(cookieClient)
		if err == nil {
			return cookieClient, resp.NetConnected == 1, err
		}

		return cookieClient, false, err
	}

	return nil, false, err
}

func (v *Ovms) connectRequest(cookieClient *request.Helper) (ovmsConnectResponse, error) {
	uri := fmt.Sprintf("http://%s:6868/api/vehicle/%s", v.server, v.vehicleId)
	var res ovmsConnectResponse

	req, err := request.New(http.MethodGet, uri, nil)
	if err == nil {
		err = cookieClient.DoJSON(req, &res)
	}

	return res, err
}

func (v *Ovms) chargeRequest(cookieClient *request.Helper) (ovmsChargeResponse, error) {
	uri := fmt.Sprintf("http://%s:6868/api/charge/%s", v.server, v.vehicleId)
	var res ovmsChargeResponse

	req, err := request.New(http.MethodGet, uri, nil)
	if err == nil {
		err = cookieClient.DoJSON(req, &res)
	}

	return res, err
}

func (v *Ovms) disconnect(cookieClient *request.Helper) error {
	uri := fmt.Sprintf("http://%s:6868/api/vehicle/%s", v.server, v.vehicleId)

	err := v.delete(uri, cookieClient)
	if err == nil {
		uri = fmt.Sprintf("http://%s:6868/api/cookie", v.server)
		return v.delete(uri, cookieClient)
	}

	return err
}

// batteryAPI provides battery-status api response
func (v *Ovms) batteryAPI() (interface{}, error) {
	var resp ovmsChargeResponse
	var ovmsConnected bool

	cookieClient, ovmsConnected, err := v.authFlow()
	if err == nil && cookieClient != nil {
		time.Sleep(3 * time.Second)
		resp, err = v.chargeRequest(cookieClient)
		for err != nil && resp.MessageAgeServer > 59 && ovmsConnected {
			time.Sleep(3 * time.Second)
			resp, err = v.chargeRequest(cookieClient)
		}
	}

	if cookieClient != nil {
		err = v.disconnect(cookieClient)
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

	return status, nil
}

var _ api.VehicleRange = (*Ovms)(nil)

// Range implements the api.VehicleRange interface
func (v *Ovms) Range() (int64, error) {
	res, err := v.chargeG()

	if res, ok := res.(ovmsChargeResponse); err == nil && ok {
		return strconv.ParseInt(res.EstimatedRange, 0, 64)
	}

	return 0, nil
}

var _ api.VehicleFinishTimer = (*Ovms)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Ovms) FinishTime() (time.Time, error) {
	res, err := v.chargeG()

	if res, ok := res.(ovmsChargeResponse); err == nil && ok {
		cef, err := strconv.ParseInt(res.ChargeEtrFull, 0, 64)
		if err == nil {
			return time.Now().Add(time.Duration(cef) * time.Minute), err
		}
	}

	return time.Time{}, api.ErrNotAvailable
}
