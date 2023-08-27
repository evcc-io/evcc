package tasmota

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Connection is the Tasmota connection
type Connection struct {
	*request.Helper
	uri, user, password string
	channel             int
	statusSNSCache      provider.Cacheable[StatusSNSResponse]
	statusSTSCache      provider.Cacheable[StatusSTSResponse]
}

// NewConnection creates a Tasmota connection
func NewConnection(uri, user, password string, channel int, cache time.Duration) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	log := util.NewLogger("tasmota")
	c := &Connection{
		Helper:   request.NewHelper(log),
		uri:      util.DefaultScheme(strings.TrimRight(uri, "/"), "http"),
		user:     user,
		password: password,
		channel:  channel,
	}

	c.Client.Transport = request.NewTripper(log, transport.Insecure())

	c.statusSNSCache = provider.ResettableCached(func() (StatusSNSResponse, error) {
		parameters := url.Values{
			"user":     []string{c.user},
			"password": []string{c.password},
			"cmnd":     []string{"Status 8"},
		}
		var res StatusSNSResponse
		err := c.GetJSON(fmt.Sprintf("%s/cm?%s", c.uri, parameters.Encode()), &res)
		return res, err
	}, cache)

	c.statusSTSCache = provider.ResettableCached(func() (StatusSTSResponse, error) {
		parameters := url.Values{
			"user":     []string{c.user},
			"password": []string{c.password},
			"cmnd":     []string{"Status 0"},
		}
		var res StatusSTSResponse
		err := c.GetJSON(fmt.Sprintf("%s/cm?%s", c.uri, parameters.Encode()), &res)
		return res, err
	}, cache)

	return c, nil
}

// channelExists checks the existence of the configured relay channel interface
func (c *Connection) ChannelExists(channel int) error {
	res, err := c.statusSTSCache.Get()
	if err != nil {
		return err
	}

	var ok bool
	switch channel {
	case 1:
		ok = res.StatusSTS.Power != "" || res.StatusSTS.Power1 != ""
	case 2:
		ok = res.StatusSTS.Power2 != ""
	case 3:
		ok = res.StatusSTS.Power3 != ""
	case 4:
		ok = res.StatusSTS.Power4 != ""
	case 5:
		ok = res.StatusSTS.Power5 != ""
	case 6:
		ok = res.StatusSTS.Power6 != ""
	case 7:
		ok = res.StatusSTS.Power7 != ""
	case 8:
		ok = res.StatusSTS.Power8 != ""
	}

	if !ok {
		return fmt.Errorf("invalid relay channel: %d", channel)
	}

	return nil
}

// Enable implements the api.Charger interface
func (c *Connection) Enable(enable bool) error {
	cmd := fmt.Sprintf("Power%d off", c.channel)
	if enable {
		cmd = fmt.Sprintf("Power%d on", c.channel)
	}

	parameters := url.Values{
		"user":     []string{c.user},
		"password": []string{c.password},
		"cmnd":     []string{cmd},
	}

	var res PowerResponse
	if err := c.GetJSON(fmt.Sprintf("%s/cm?%s", c.uri, parameters.Encode()), &res); err != nil {
		return err
	}

	var on bool
	switch c.channel {
	case 2:
		on = strings.ToUpper(res.Power2) == "ON"
	case 3:
		on = strings.ToUpper(res.Power3) == "ON"
	case 4:
		on = strings.ToUpper(res.Power4) == "ON"
	case 5:
		on = strings.ToUpper(res.Power5) == "ON"
	case 6:
		on = strings.ToUpper(res.Power6) == "ON"
	case 7:
		on = strings.ToUpper(res.Power7) == "ON"
	case 8:
		on = strings.ToUpper(res.Power8) == "ON"
	default:
		on = strings.ToUpper(res.Power) == "ON" || strings.ToUpper(res.Power1) == "ON"
	}

	c.statusSNSCache.Reset()
	c.statusSTSCache.Reset()

	switch {
	case enable && !on:
		return errors.New("switchOn failed")
	case !enable && on:
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

// Enabled implements the api.Charger interface
func (c *Connection) Enabled() (bool, error) {
	res, err := c.statusSTSCache.Get()
	if err != nil {
		return false, err
	}

	switch c.channel {
	case 2:
		return strings.ToUpper(res.StatusSTS.Power2) == "ON", err
	case 3:
		return strings.ToUpper(res.StatusSTS.Power3) == "ON", err
	case 4:
		return strings.ToUpper(res.StatusSTS.Power4) == "ON", err
	case 5:
		return strings.ToUpper(res.StatusSTS.Power5) == "ON", err
	case 6:
		return strings.ToUpper(res.StatusSTS.Power6) == "ON", err
	case 7:
		return strings.ToUpper(res.StatusSTS.Power7) == "ON", err
	case 8:
		return strings.ToUpper(res.StatusSTS.Power8) == "ON", err
	default:
		return strings.ToUpper(res.StatusSTS.Power) == "ON" || strings.ToUpper(res.StatusSTS.Power1) == "ON", err
	}
}

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	res, err := c.statusSNSCache.Get()
	if err != nil {
		return 0, err
	}
	return res.StatusSNS.Energy.Power.Channel(c.channel)
}

// TotalEnergy implements the api.MeterEnergy interface
func (c *Connection) TotalEnergy() (float64, error) {
	res, err := c.statusSNSCache.Get()
	return res.StatusSNS.Energy.Total, err
}

// SmlPower provides the sml sensor power
func (c *Connection) SmlPower() (float64, error) {
	res, err := c.statusSNSCache.Get()
	return float64(res.StatusSNS.SML.PowerCurr), err
}

// SmlTotalEnergy provides the sml sensor total import energy
func (c *Connection) SmlTotalEnergy() (float64, error) {
	res, err := c.statusSNSCache.Get()
	return res.StatusSNS.SML.TotalIn, err
}
