package charger

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// Sonoff device rpc api
// https://help.sonoff.tech/docs/API_Welcome

type sonoffRequest struct {
	Id     int    `json:"id"`
	Src    string `json:"src"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type sonoffChannelParams struct {
	Id int `json:"id"`
}

type sonoffSetParams struct {
	Id int  `json:"id"`
	On bool `json:"on"`
}

// sonoffSwitchStatus is the Switch.GetStatus result
// https://help.sonoff.tech/docs/API_Switch
type sonoffSwitchStatus struct {
	On bool `json:"on"`
}

// sonoffMeterStatus is the Meter.GetStatus result, scaled by 100 (energy in 0.01kWh)
// https://help.sonoff.tech/docs/API_Meter
type sonoffMeterStatus struct {
	Voltage     float64 `json:"voltage"`
	Current     float64 `json:"current"`
	Power       float64 `json:"power"`
	TotalEnergy float64 `json:"total_energy"`
}

// Sonoff charger implementation
type Sonoff struct {
	implement.Caps
	*switchSocket
	*request.Helper
	uri     string
	channel int
	statusG util.Cacheable[sonoffSwitchStatus]
	meterG  util.Cacheable[sonoffMeterStatus]
}

func init() {
	registry.Add("sonoff", NewSonoffFromConfig)
}

// NewSonoffFromConfig creates a Sonoff charger from generic config
func NewSonoffFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		embed        `mapstructure:",squash"`
		URI          string
		User         string
		Password     string
		Channel      int
		StandbyPower float64
		Cache        time.Duration
	}{
		User:    "admin",
		Channel: 1,
		Cache:   time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSonoff(cc.embed, cc.URI, cc.User, cc.Password, cc.Channel, cc.StandbyPower, cc.Cache)
}

// NewSonoff creates a Sonoff charger
func NewSonoff(embed embed, uri, user, password string, channel int, standbypower float64, cache time.Duration) (api.Charger, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	log := util.NewLogger("sonoff").Redact(user, password)

	c := &Sonoff{
		Caps:    implement.New(),
		Helper:  request.NewHelper(log),
		uri:     fmt.Sprintf("%s/rpc", util.DefaultScheme(strings.TrimRight(uri, "/"), "http")),
		channel: channel,
	}

	// rfc7616 digest authentication
	// https://help.sonoff.tech/docs/API_Authentication
	if password != "" {
		c.Client.Transport = digest.NewTransport(user, password, c.Client.Transport)
	}

	c.statusG = util.ResettableCached(func() (sonoffSwitchStatus, error) {
		return sonoffCall[sonoffSwitchStatus](c, "Switch.GetStatus", sonoffChannelParams{Id: channel})
	}, cache)

	c.meterG = util.ResettableCached(func() (sonoffMeterStatus, error) {
		return sonoffCall[sonoffMeterStatus](c, "Meter.GetStatus", sonoffChannelParams{Id: channel})
	}, cache)

	// validate switch channel
	if _, err := c.statusG.Get(); err != nil {
		return nil, err
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.currentPower, standbypower)

	// metering is optional, devices without meter component fail the call
	if _, err := c.meterG.Get(); err == nil {
		implement.Has(c, implement.MeterEnergy(c.totalEnergy))
		implement.Has(c, implement.PhaseCurrents(c.currents))
		implement.Has(c, implement.PhaseVoltages(c.voltages))
	}

	return c, nil
}

// sonoffCall executes a single rpc method and returns its result
func sonoffCall[T any](c *Sonoff, method string, params any) (T, error) {
	var res struct {
		Result T `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	data := sonoffRequest{
		Id:     c.channel,
		Src:    "evcc",
		Method: method,
		Params: params,
	}

	req, err := request.New(http.MethodPost, c.uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return res.Result, err
	}

	if err := c.DoJSON(req, &res); err != nil {
		return res.Result, err
	}

	if res.Error != nil {
		return res.Result, fmt.Errorf("%s: %s (%d)", method, res.Error.Message, res.Error.Code)
	}

	return res.Result, nil
}

// Enabled implements the api.Charger interface
func (c *Sonoff) Enabled() (bool, error) {
	res, err := c.statusG.Get()
	return res.On, err
}

// Enable implements the api.Charger interface
func (c *Sonoff) Enable(enable bool) error {
	c.statusG.Reset()

	_, err := sonoffCall[struct{}](c, "Switch.Set", sonoffSetParams{Id: c.channel, On: enable})
	return err
}

func (c *Sonoff) currentPower() (float64, error) {
	res, err := c.meterG.Get()
	return res.Power / 100, err
}

func (c *Sonoff) totalEnergy() (float64, error) {
	res, err := c.meterG.Get()
	return res.TotalEnergy / 100, err
}

func (c *Sonoff) currents() (float64, float64, float64, error) {
	res, err := c.meterG.Get()
	return res.Current / 100, 0, 0, err
}

func (c *Sonoff) voltages() (float64, float64, float64, error) {
	res, err := c.meterG.Get()
	return res.Voltage / 100, 0, 0, err
}
