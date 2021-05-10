package vehicle

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

type ovmsChargeResponse struct {
	BattVoltage         string `json:"battvoltage"`
	Cac100              string `json:"cac100"`
	CarAwake            int    `json:"carawake"`
	CarOn               int    `json:"caron"`
	ChargeEstimate      string `json:"charge_estimate"`
	ChargeEtrFull       string `json:"charge_etr_full"`
	ChargeEtrLimit      string `json:"charge_etr_limit"`
	ChargeEtrRange      string `json:"charge_etr_range"`
	ChargeEtrSoc        string `json:"charge_etr_soc"`
	ChargeLimitRange    string `json:"charge_limit_range"`
	ChargeLimitSoc      string `json:"charge_limit_soc"`
	ChargeB4            string `json:"chargeb4"`
	ChargeCurrent       string `json:"chargecurrent"`
	ChargeDuration      string `json:"chargeduration"`
	ChargeEnergy        string `json:"chargekwh"`
	ChargeLimit         string `json:"chargelimit"`
	ChargePower         string `json:"chargepower"`
	ChargeStartTime     string `json:"chargestarttime"`
	ChargeState         string `json:"chargestate"`
	ChargeSubState      string `json:"chargesubstate"`
	ChargeTimerMode     string `json:"chargetimermode"`
	ChargeTimerStale    string `json:"chargetimerstale"`
	ChargeTape          string `json:"chargetype"`
	Charging            int    `json:"charging"`
	Charging12v         int    `json:"charging_12v"`
	CoolDownActive      string `json:"cooldown_active"`
	CoolDownBattery     string `json:"cooldown_tbattery"`
	CoolDownTimeLimit   string `json:"cooldown_timelimit"`
	ChargePortOpen      int    `json:"cp_dooropen"`
	EstimatedRange      string `json:"estimatedrange"`
	IdealRange          string `json:"idealrange"`
	IdealRangeMax       string `json:"idealrange_max"`
	LineVoltage         string `json:"linevoltage"`
	Mode                string `json:"mode"`
	PilotPresent        int    `json:"pilotpresent"`
	Soc                 string `json:"soc"`
	Soh                 string `json:"soh"`
	StaleAmbient        string `json:"staleambient"`
	StaleTemps          string `json:"staletemps"`
	TemperaturAmbient   string `json:"temperature_ambient"`
	TemperaturBattery   string `json:"temperature_battery"`
	TemperaturCharger   string `json:"temperature_charger"`
	TemperaturMotor     string `json:"temperature_motor"`
	TemperaturPem       string `json:"temperature_pem"`
	Units               string `json:"units"`
	Vehicle12v          string `json:"vehicle12v"`
	Vehicle12vCurrent   string `json:"vehicle12v_current"`
	Vehicle12vReference string `json:"vehicle12v_ref"`
}

// OVMS is an api.Vehicle implementation for dexters-web server requests
type Ovms struct {
	*embed
	*request.Helper
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
		Helper:    request.NewHelper(log),
		user:      cc.User,
		password:  cc.Password,
		vehicleId: cc.VehicleID,
		server:    cc.Server,
	}

	v.chargeG = provider.NewCached(v.batteryAPI, cc.Cache).InterfaceGetter()

	return v, nil
}

func (r *Ovms) GetCookie(url string) ([]*http.Cookie, error) {
	resp, err := r.Get(url)
	if err == nil {
		return resp.Cookies(), nil
	}
	return nil, err
}

func (v *Ovms) request(uri string, cookie *http.Cookie) (*http.Request, error) {
	req, err := request.New(http.MethodGet, uri, nil)
	if err == nil {
		req.AddCookie(cookie)
	}

	return req, err
}

func (v *Ovms) sessionCookie() (*http.Cookie, error) {
	uri := fmt.Sprintf("http://%s:6868/api/cookie?username=%s&password=%s", v.server, v.user, v.password)

	cookies, err := v.GetCookie(uri)

	var cookie *http.Cookie
	if err == nil && len(cookies) > 0 {
		cookie = cookies[0]
	}

	return cookie, err
}

func (v *Ovms) chargeRequest(uri string, cookie *http.Cookie) (ovmsChargeResponse, error) {
	var res ovmsChargeResponse
	req, err := v.request(uri, cookie)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

// batteryAPI provides battery-status api response
func (v *Ovms) batteryAPI() (interface{}, error) {
	uri := fmt.Sprintf("http://%s:6868/api/charge/%s", v.server, v.vehicleId)
	var res ovmsChargeResponse

	cookie, err := v.sessionCookie()
	if err == nil {
		res, err = v.chargeRequest(uri, cookie)
	}

	return res, err
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
		if res.ChargeState != "stopped" {
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

var _ api.VehicleClimater = (*Ovms)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Ovms) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	return false, 0, 0, api.ErrNotAvailable
}
