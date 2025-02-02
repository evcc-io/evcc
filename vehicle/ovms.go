package vehicle

import (
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/ovms"
	"golang.org/x/net/publicsuffix"
)

// OVMS is an api.Vehicle implementation for dexters-web server requests
type Ovms struct {
	*embed
	*request.Helper
	user, password    string
	vehicleId, server string
	cache             time.Duration
	isOnline          bool
	chargeG           func() (ovms.ChargeResponse, error)
	statusG           func() (ovms.StatusResponse, error)
	locationG         func() (ovms.LocationResponse, error)
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

	v.Jar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	return v, nil
}

func (v *Ovms) loginToServer() (err error) {
	uri := fmt.Sprintf("https://%s:6869/api/cookie?username=%s&password=%s", v.server, url.QueryEscape(v.user), url.QueryEscape(v.password))

	var resp *http.Response
	if resp, err = v.Get(uri); err == nil {
		resp.Body.Close()
	}

	return err
}

func (v *Ovms) uri(path string) string {
	return fmt.Sprintf("https://%s/api/%s/%s", net.JoinHostPort(v.server, "6869"), path, v.vehicleId)
}

func (v *Ovms) connectRequest() (ovms.ConnectResponse, error) {
	var res ovms.ConnectResponse
	err := v.GetJSON(v.uri("vehicle"), &res)
	return res, err
}

func (v *Ovms) chargeRequest() (ovms.ChargeResponse, error) {
	var res ovms.ChargeResponse
	err := v.GetJSON(v.uri("charge"), &res)
	return res, err
}

func (v *Ovms) statusRequest() (ovms.StatusResponse, error) {
	var res ovms.StatusResponse
	err := v.GetJSON(v.uri("status"), &res)
	return res, err
}

func (v *Ovms) locationRequest() (ovms.LocationResponse, error) {
	var res ovms.LocationResponse
	err := v.GetJSON(v.uri("location"), &res)
	return res, err
}

func (v *Ovms) authFlow() error {
	var resp ovms.ConnectResponse
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
func (v *Ovms) batteryAPI() (ovms.ChargeResponse, error) {
	var resp ovms.ChargeResponse

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
func (v *Ovms) statusAPI() (ovms.StatusResponse, error) {
	var resp ovms.StatusResponse

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
func (v *Ovms) locationAPI() (ovms.LocationResponse, error) {
	var resp ovms.LocationResponse

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

const kmPerMile = 1.609344

// Range implements the api.VehicleRange interface
func (v *Ovms) Range() (int64, error) {
	res, err := v.chargeG()
	if res.Units == ovms.UnitMiles {
		res.EstimatedRange = int64(float64(res.EstimatedRange) * kmPerMile)
	}
	return res.EstimatedRange, err
}

var _ api.VehicleOdometer = (*Ovms)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Ovms) Odometer() (float64, error) {
	res, err := v.statusG()
	if res.Units == ovms.UnitMiles {
		res.Odometer *= kmPerMile
	}
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
