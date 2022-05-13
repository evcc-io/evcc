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
	Channel             int
}

// NewConnection creates Tasmota charger
func NewConnection(uri, user, password string, channel int) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	// Default Tasmota relay channel 1
	if channel == 0 {
		channel = 1
	}

	if channel < 1 || channel > 4 {
		return nil, errors.New("invalid relais channel (allowed range: 1-4)")
	}

	log := util.NewLogger("tasmota")
	c := &Connection{
		Helper:   request.NewHelper(log),
		uri:      util.DefaultScheme(strings.TrimRight(uri, "/"), "http"),
		user:     user,
		password: password,
		Channel:  channel,
	}

	c.Client.Transport = request.NewTripper(log, transport.Insecure())

	return c, nil
}

// ExecCmd executes a Tasmota api command and provides the response
func (d *Connection) ExecCmd(cmd string, res interface{}) error {
	parameters := url.Values{
		"user":     []string{d.user},
		"password": []string{d.password},
		"cmnd":     []string{cmd},
	}

	err := d.GetJSON(fmt.Sprintf("%s/cm?%s", d.uri, parameters.Encode()), res)
	if err != nil {
		return err
	}

	return nil
}

// CurrentPower provides current power consumption
func (d *Connection) CurrentPower() (float64, error) {
	var res *StatusSNSResponse
	err := d.ExecCmd("Status 8", &res)
	if err != nil {
		return 0, err
	}

	return float64(res.StatusSNS.Energy.Power), nil
}

// TotalEnergy implements the api.MeterEnergy interface
func (d *Connection) TotalEnergy() (float64, error) {
	var res *StatusSNSResponse
	err := d.ExecCmd("Status 8", &res)
	if err != nil {
		return 0, err
	}

	return float64(res.StatusSNS.Energy.Total), nil
}
