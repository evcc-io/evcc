package tasmota

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Connection is the Tasmota connection
type Connection struct {
	*request.Helper
	uri, user, password string
	channel             int
}

// NewConnection creates a Tasmota connection
func NewConnection(uri, user, password string, channel int) (*Connection, error) {
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
func (d *Connection) CurrentPower() (float64, error) {
	var res StatusSNSResponse
	if err := d.ExecCmd("Status 8", &res); err != nil {
		return 0, err
	}
	return res.StatusSNS.Energy.Power.Channel(d.channel)
}

// TotalEnergy implements the api.MeterEnergy interface
func (d *Connection) TotalEnergy() (float64, error) {
	var res StatusSNSResponse
	err := d.ExecCmd("Status 8", &res)
	return res.StatusSNS.Energy.Total, err
}

// SmlPower provides the sml sensor power
func (d *Connection) SmlPower() (float64, error) {
	var res StatusSNSResponse
	err := d.ExecCmd("Status 8", &res)
	return float64(res.StatusSNS.SML.PowerCurr), err
}

// SmlTotalEnergy provides the sml sensor total import energy
func (d *Connection) SmlTotalEnergy() (float64, error) {
	var res StatusSNSResponse
	err := d.ExecCmd("Status 8", &res)
	return res.StatusSNS.SML.TotalIn, err
}
