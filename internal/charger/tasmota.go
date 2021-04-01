package charger

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// Tasmota project homepage
// https://tasmota.github.io/docs/
// Supported devices:
// https://templates.blakadder.com/

// Tasmota charger implementation
type Tasmota struct {
	*request.Helper
	uri          string
	parameters   url.Values
	standbypower float64
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
	parameters := url.Values{
		"user":     []string{user},
		"password": []string{password},
		"cmnd":     []string{""},
	}
	c := &Tasmota{
		Helper:       request.NewHelper(log),
		uri:          fmt.Sprintf("%s/cm"+"?", strings.TrimRight(uri, "/")),
		parameters:   parameters,
		standbypower: standbypower,
	}
	c.Client.Transport = request.NewTripper(log, request.InsecureTransport())
	return c, nil
}

// Enabled implements the Charger.Enabled interface
func (c *Tasmota) Enabled() (bool, error) {
	// tStatus is the Tasmota Status 0 command Status response
	// https://tasmota.github.io/docs/Commands/#with-web-requests
	var tStatus struct {
		Status struct {
			Module       int
			DeviceName   string
			FriendlyName []string
			Topic        string
			ButtonTopic  string
			Power        int
			PowerOnState int
			LedState     int
			LedMask      string
			SaveData     int
			SaveState    int
			SwitchTopic  string
			SwitchMode   []int
			ButtonRetain int
			SwitchRetain int
			SensorRetain int
			PowerRetain  int
			InfoRetain   int
			StateRetain  int
		}
	}
	// Execute Tasmota Status 0 command
	c.parameters.Set("cmnd", "Status 0")
	err := c.GetJSON(c.uri+c.parameters.Encode(), &tStatus)
	return int(1) == tStatus.Status.Power, err
}

// Enable implements the Charger.Enable interface
func (c *Tasmota) Enable(enable bool) error {
	// tPower is the Tasmota Power command response
	// https://tasmota.github.io/docs/Commands/#with-web-requests
	var tPower struct {
		POWER string
	}
	cmnd := "Power off"
	if enable {
		cmnd = "Power on"
	}
	// Execute Tasmota Power on/off command
	c.parameters.Set("cmnd", cmnd)
	err := c.GetJSON(c.uri+c.parameters.Encode(), &tPower)
	switch {
	case err != nil:
		return err
	case enable && tPower.POWER != "ON":
		return errors.New("switchOn failed")
	case !enable && tPower.POWER != "OFF":
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *Tasmota) MaxCurrent(current int64) error {
	return nil
}

// Status implements the Charger.Status interface
func (c *Tasmota) Status() (api.ChargeStatus, error) {
	power, err := c.CurrentPower()
	switch {
	case power > 0:
		return api.StatusC, err
	default:
		return api.StatusB, err
	}
}

var _ api.Meter = (*Tasmota)(nil)

// CurrentPower implements the api.Meter interface
func (c *Tasmota) CurrentPower() (float64, error) {
	// tStatusSNS is the Tasmota Status 8 command response
	// https://tasmota.github.io/docs/Commands/#power-monitoring
	var tStatusSNS struct {
		StatusSNS struct {
			Time   string
			Energy struct {
				TotalStartTime string
				Total          float64
				Yesterday      float64
				Today          float64
				Power          int
				ApparentPower  int
				ReactivePower  int
				Factor         float64
				Voltage        int
				Current        float64
			}
		}
	}
	// Execute Tasmota Status 8 command
	c.parameters.Set("cmnd", "Status 8")
	err := c.GetJSON(c.uri+c.parameters.Encode(), &tStatusSNS)

	if err != nil {
		return math.NaN(), err
	}
	power := float64(tStatusSNS.StatusSNS.Energy.Power)
	// ignore standby power
	if power < c.standbypower {
		power = 0
	}
	return power, err
}
