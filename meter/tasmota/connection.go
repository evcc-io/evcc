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
	channels            []int
	statusSnsG          provider.Cacheable[StatusSNSResponse]
	statusStsG          provider.Cacheable[StatusSTSResponse]
}

// NewConnection creates a Tasmota connection
func NewConnection(uri, user, password string, channels []int, cache time.Duration) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	used := make(map[int]bool)
	for _, c := range channels {
		if c < 1 || c > 8 {
			return nil, fmt.Errorf("invalid channel: %d", c)
		}
		if used[c] {
			return nil, fmt.Errorf("duplicate channel: %d", c)
		}
		used[c] = true
	}

	log := util.NewLogger("tasmota")
	c := &Connection{
		Helper:   request.NewHelper(log),
		uri:      util.DefaultScheme(strings.TrimRight(uri, "/"), "http"),
		user:     user,
		password: password,
		channels: channels,
	}

	c.Client.Transport = request.NewTripper(log, transport.Insecure())

	c.statusSnsG = provider.ResettableCached(func() (StatusSNSResponse, error) {
		parameters := url.Values{
			"user":     []string{c.user},
			"password": []string{c.password},
			"cmnd":     []string{"Status 8"},
		}
		var res StatusSNSResponse
		err := c.GetJSON(fmt.Sprintf("%s/cm?%s", c.uri, parameters.Encode()), &res)
		return res, err
	}, cache)

	c.statusStsG = provider.ResettableCached(func() (StatusSTSResponse, error) {
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
func (c *Connection) RelayExists() error {
	res, err := c.statusStsG.Get()
	if err != nil {
		return err
	}

	var ok bool
	for _, channel := range c.channels {
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
	}

	return nil
}

// Enable implements the api.Charger interface
func (c *Connection) Enable(enable bool) error {
	for _, channel := range c.channels {
		cmd := fmt.Sprintf("Power%d off", channel)
		if enable {
			cmd = fmt.Sprintf("Power%d on", channel)
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

		var enabled bool
		switch channel {
		case 2:
			enabled = strings.ToUpper(res.Power2) == "ON"
		case 3:
			enabled = strings.ToUpper(res.Power3) == "ON"
		case 4:
			enabled = strings.ToUpper(res.Power4) == "ON"
		case 5:
			enabled = strings.ToUpper(res.Power5) == "ON"
		case 6:
			enabled = strings.ToUpper(res.Power6) == "ON"
		case 7:
			enabled = strings.ToUpper(res.Power7) == "ON"
		case 8:
			enabled = strings.ToUpper(res.Power8) == "ON"
		default:
			enabled = strings.ToUpper(res.Power) == "ON" || strings.ToUpper(res.Power1) == "ON"
		}

		switch {
		case enable && !enabled:
			return errors.New("switchOn failed")
		case !enable && enabled:
			return errors.New("switchOff failed")
		}
	}

	c.statusSnsG.Reset()
	c.statusStsG.Reset()

	return nil
}

// Enabled implements the api.Charger interface
func (c *Connection) Enabled() (bool, error) {
	res, err := c.statusStsG.Get()
	if err != nil {
		return false, err
	}

	var enabled bool
	for _, channel := range c.channels {
		switch channel {
		case 2:
			enabled = strings.ToUpper(res.StatusSTS.Power2) == "ON"
		case 3:
			enabled = strings.ToUpper(res.StatusSTS.Power3) == "ON"
		case 4:
			enabled = strings.ToUpper(res.StatusSTS.Power4) == "ON"
		case 5:
			enabled = strings.ToUpper(res.StatusSTS.Power5) == "ON"
		case 6:
			enabled = strings.ToUpper(res.StatusSTS.Power6) == "ON"
		case 7:
			enabled = strings.ToUpper(res.StatusSTS.Power7) == "ON"
		case 8:
			enabled = strings.ToUpper(res.StatusSTS.Power8) == "ON"
		default:
			enabled = strings.ToUpper(res.StatusSTS.Power) == "ON" || strings.ToUpper(res.StatusSTS.Power1) == "ON"
		}
	}
	return enabled, nil
}

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	s, err := c.statusSnsG.Get()
	if err != nil {
		return 0, err
	}
	var res float64
	for _, channel := range c.channels {
		power, err := s.StatusSNS.Energy.Power.Channel(channel)
		if err != nil {
			return 0, err
		}
		res += power
	}
	return res, nil
}

// TotalEnergy implements the api.MeterEnergy interface
func (c *Connection) TotalEnergy() (float64, error) {
	res, err := c.statusSnsG.Get()
	return res.StatusSNS.Energy.Total, err
}

// Currents implements the api.PhaseCurrents interface
func (c *Connection) Currents() (float64, float64, float64, error) {
	s, err := c.statusSnsG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i, c := range c.channels {
		res[i], err = s.StatusSNS.Energy.Current.Channel(c)
		if err != nil {
			return 0, 0, 0, err
		}
	}

	return res[0], res[1], res[2], err
}

// Voltages implements the api.PhaseVoltages interface
func (c *Connection) Voltages() (float64, float64, float64, error) {
	s, err := c.statusSnsG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i, c := range c.channels {
		res[i], err = s.StatusSNS.Energy.Voltage.Channel(c)
		if err != nil {
			return 0, 0, 0, err
		}
	}

	return res[0], res[1], res[2], err
}

// SmlPower provides the sml sensor power
func (c *Connection) SmlPower() (float64, error) {
	res, err := c.statusSnsG.Get()
	return float64(res.StatusSNS.SML.PowerCurr), err
}

// SmlTotalEnergy provides the sml sensor total import energy
func (c *Connection) SmlTotalEnergy() (float64, error) {
	res, err := c.statusSnsG.Get()
	return res.StatusSNS.SML.TotalIn, err
}
