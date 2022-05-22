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
}

// NewConnection creates a Tasmota connection
func NewConnection(uri, user, password string) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	log := util.NewLogger("tasmota")
	c := &Connection{
		Helper:   request.NewHelper(log),
		uri:      util.DefaultScheme(strings.TrimRight(uri, "/"), "http"),
		user:     user,
		password: password,
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
