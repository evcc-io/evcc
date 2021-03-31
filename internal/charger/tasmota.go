package charger

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// Tasmota project homepage
// https://tasmota.github.io/docs/
// Supported devices:
// https://templates.blakadder.com/

// tasmotaStatusSNSResponse is the Status 8 command response
// https://tasmota.github.io/docs/Commands/#power-monitoring
type tasmotaStatusSNSResponse struct {
	StatusSNS struct {
		Time   string `json:"Time"`
		Energy struct {
			TotalStartTime string  `json:"TotalStartTime"`
			Total          float64 `json:"Total"`
			Yesterday      float64 `json:"Yesterday"`
			Today          float64 `json:"Today"`
			Power          int     `json:"Power"`
			ApparentPower  int     `json:"ApparentPower"`
			ReactivePower  int     `json:"ReactivePower"`
			Factor         float64 `json:"Factor"`
			Voltage        int     `json:"Voltage"`
			Current        float64 `json:"Current"`
		}
	}
}

var tStatusSNS tasmotaStatusSNSResponse

// tasmotaStatusResponse is the Status 0 command Status response
type tasmotaStatusResponse struct {
	Status struct {
		Module       int      `json:"Module"`
		DeviceName   string   `json:"DeviceName"`
		FriendlyName []string `json:"FriendlyName"`
		Topic        string   `json:"Topic"`
		ButtonTopic  string   `json:"ButtonTopic"`
		Power        int      `json:"Power"`
		PowerOnState int      `json:"PowerOnState"`
		LedState     int      `json:"LedState"`
		LedMask      string   `json:"LedMask"`
		SaveData     int      `json:"SaveData"`
		SaveState    int      `json:"SaveState"`
		SwitchTopic  string   `json:"SwitchTopic"`
		SwitchMode   []int    `json:"SwitchMode"`
		ButtonRetain int      `json:"ButtonRetain"`
		SwitchRetain int      `json:"SwitchRetain"`
		SensorRetain int      `json:"SensorRetain"`
		PowerRetain  int      `json:"PowerRetain"`
		InfoRetain   int      `json:"InfoRetain"`
		StateRetain  int      `json:"StateRetain"`
	}
}

var tStatus tasmotaStatusResponse

// tasmotaPowerResponse is the Power command response
// https://tasmota.github.io/docs/Commands/#with-web-requests
type tasmotaPowerResponse struct {
	POWER string `json:"POWER"`
}

var tPower tasmotaPowerResponse

// Tasmota charger implementation
type Tasmota struct {
	*request.Helper
	uri, user, password string
	standbypower        float64
}

func init() {
	registry.Add("tasmota", NewTasmotaFromConfig)
}

// NewTasmotaFromConfig creates a Tasmota charger from generic config
func NewTasmotaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI          string
		User         string
		Password     string
		StandbyPower float64
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewTasmota(cc.URI, cc.User, cc.Password, cc.StandbyPower)
}

// NewTasmota creates Tasmota charger
func NewTasmota(uri, user, password string, standbypower float64) (*Tasmota, error) {
	log := util.NewLogger("tasmota")

	c := &Tasmota{
		Helper:       request.NewHelper(log),
		uri:          strings.TrimRight(uri, "/"),
		user:         user,
		password:     password,
		standbypower: standbypower,
	}

	c.Client.Transport = request.NewTripper(log, request.InsecureTransport())

	return c, nil
}

// Enable implements the Charger.Enable interface
func (c *Tasmota) Enable(enable bool) error {
	cmd := "Power off"
	if enable {
		cmd = "Power on"
	}

	// state ON/OFF/Error - Tasmota Switch state off/on (empty if unkown or error)
	resp, err := c.execTasmotaCmd(cmd)

	switch {
	case err != nil:
		return err
	case enable && resp != "ON":
		return errors.New("switchOn failed")
	case !enable && resp != "OFF":
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

// Enabled implements the Charger.Enabled interface
func (c *Tasmota) Enabled() (bool, error) {
	// state 0/1 - Tasmota Switch state off/on (empty if unkown or error)
	resp, err := c.execTasmotaCmd("Status 0")

	var state int64
	if err == nil {
		state, err = strconv.ParseInt(resp, 10, 32)
	}

	return state == 1, err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *Tasmota) MaxCurrent(current int64) error {
	return nil
}

var _ api.Meter = (*Tasmota)(nil)

// CurrentPower implements the api.Meter interface
func (c *Tasmota) CurrentPower() (float64, error) {
	// https://tasmota.github.io/docs/Commands/#power-monitoring
	// stat/pow1/STATUS8 = {"StatusSNS":{"Time":"2018-11-15T08:54:18","ENERGY":{"TotalStartTime":"2018-11-14T18:39:40","Total":6.404,"Yesterday":5.340,"Today":1.064,"Power":2629,"ApparentPower":2645,"ReactivePower":288,"Factor":0.99,"Voltage":226,"Current":11.677}}}
	resp, err := c.execTasmotaCmd("Status 8")

	var power float64
	if err == nil {
		power, err = strconv.ParseFloat(resp, 64)
	}

	// ignore standby power
	if power < c.standbypower {
		power = 0
	}

	return power, err
}

// Status implements the Charger.Status interface
func (c *Tasmota) Status() (api.ChargeStatus, error) {

	// power value in W (current switch power, refresh aproximately every 2 minutes)
	var power float64
	resp, err := c.execTasmotaCmd("Status 8")
	if err == nil {
		power, err = strconv.ParseFloat(resp, 64)
	}

	switch {
	case power <= c.standbypower:
		return api.StatusB, err
	default:
		return api.StatusC, err
	}
}

// execTasmotaCmd executes Tasmota commands via Web Request
func (c *Tasmota) execTasmotaCmd(cm string) (string, error) {

	// Create Tasmota Web Request: https://tasmota.github.io/docs/Commands/#with-web-requests
	uri := fmt.Sprintf("%s/cm", c.uri)
	parameters := url.Values{
		"user":     []string{c.user},
		"password": []string{c.password},
		"cmnd":     []string{cm},
	}

	switch {
	case cm == "Status 0":
		err := c.GetJSON(uri+"?"+parameters.Encode(), &tStatus)
		return strconv.Itoa(tStatus.Status.Power), err
	case cm == "Status 8":
		err := c.GetJSON(uri+"?"+parameters.Encode(), &tStatusSNS)
		return strconv.Itoa(tStatusSNS.StatusSNS.Energy.Power), err
	case cm == "Power on" || cm == "Power off":
		err := c.GetJSON(uri+"?"+parameters.Encode(), &tPower)
		return tPower.POWER, err
	default:
		return "", errors.New("unknown cm: " + cm)
	}
}
