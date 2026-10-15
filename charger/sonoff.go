package charger

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/charger/sonoff"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Sonoff charger implementation
type Sonoff struct {
	implement.Caps
	*switchSocket
	*request.Helper
	uri     string
	channel int
	statusG util.Cacheable[sonoff.SwitchStatus]
	meterG  util.Cacheable[sonoff.MeterStatus]
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
		c.Client.Transport = transport.Digest(user, password, c.Client.Transport)
	}

	c.statusG = util.ResettableCached(func() (sonoff.SwitchStatus, error) {
		var res sonoff.SwitchStatus
		err := c.call("Switch.GetStatus", sonoff.ChannelParams{Id: channel}, &res)
		return res, err
	}, cache)

	c.meterG = util.ResettableCached(func() (sonoff.MeterStatus, error) {
		var res sonoff.MeterStatus
		err := c.call("Meter.GetStatus", sonoff.ChannelParams{Id: channel}, &res)
		return res, err
	}, cache)

	// validate switch channel
	if _, err := c.statusG.Get(); err != nil {
		return nil, err
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.currentPower, standbypower)

	// metering is optional, devices without meter component fail the call
	if _, err := c.meterG.Get(); err == nil {
		implement.Has(c, implement.MeterEnergy(c.totalEnergy))
	}

	return c, nil
}

// call executes a single rpc method and decodes its result into res
func (c *Sonoff) call(method string, params, res any) error {
	data := sonoff.Request{
		Id:     c.channel,
		Src:    "evcc",
		Method: method,
		Params: params,
	}

	req, err := request.New(http.MethodPost, c.uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	var resp sonoff.Response
	if err := c.DoJSON(req, &resp); err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("%s: %w", method, resp.Error)
	}

	if res == nil {
		return nil
	}

	return json.Unmarshal(resp.Result, res)
}

// Enabled implements the api.Charger interface
func (c *Sonoff) Enabled() (bool, error) {
	res, err := c.statusG.Get()
	return res.On, err
}

// Enable implements the api.Charger interface
func (c *Sonoff) Enable(enable bool) error {
	c.statusG.Reset()

	return c.call("Switch.Set", sonoff.SetParams{Id: c.channel, On: enable}, nil)
}

func (c *Sonoff) currentPower() (float64, error) {
	res, err := c.meterG.Get()
	return res.Power / 100, err
}

func (c *Sonoff) totalEnergy() (float64, error) {
	res, err := c.meterG.Get()
	return res.TotalEnergy / 100, err
}
