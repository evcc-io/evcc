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
			"user":     []string{user},
			"password": []string{password},
			"cmnd":     []string{"Status 8"},
		}
		var res StatusSNSResponse
		err := c.GetJSON(fmt.Sprintf("%s/cm?%s", uri, parameters.Encode()), &res)
		return res, err
	}, cache)

	return c, nil
}

// ExecCmd executes an api command and provides the response
func (d *Connection) ExecCmd(cmd string, res interface{}) error {
	parameters := url.Values{
		"user":     []string{d.user},
		"password": []string{d.password},
		"cmnd":     []string{cmd},
	}

	return d.GetJSON(fmt.Sprintf("%s/cm?%s", d.uri, parameters.Encode()), res)
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
