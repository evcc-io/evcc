package charger

import (
	"errors"
	"math"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/charger/tasmota"
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
		uri:          uri,
		user:         user,
		password:     password,
		standbypower: standbypower,
	}

	c.Client.Transport = request.NewTripper(log, request.InsecureTransport())

	return c, nil
}

// Enabled implements the Charger.Enabled interface
func (c *Tasmota) Enabled() (bool, error) {
	var tStatus tasmota.StatusResponse

	// Execute Tasmota Status 0 command
	err := c.GetJSON(tasmota.CreateRequest(c.uri, c.user, c.password, "Status 0"), &tStatus)

	return int(1) == tStatus.Status.Power, err
}

// Enable implements the Charger.Enable interface
func (c *Tasmota) Enable(enable bool) error {
	var tPower tasmota.PowerResponse

	cmnd := "Power off"
	if enable {
		cmnd = "Power on"
	}

	// Execute Tasmota Power on/off command
	err := c.GetJSON(tasmota.CreateRequest(c.uri, c.user, c.password, cmnd), &tPower)

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
	var tStatusSNS tasmota.StatusSNSResponse

	// Execute Tasmota Status 8 command
	err := c.GetJSON(tasmota.CreateRequest(c.uri, c.user, c.password, "Status 8"), &tStatusSNS)

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
