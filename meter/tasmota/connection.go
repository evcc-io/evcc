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

	return d.GetJSON(fmt.Sprintf("%s/cm?%s", d.uri, parameters.Encode()), res)
}

// CurrentPower implements the api.Meter interface
func (d *Connection) CurrentPower() (float64, error) {
	var res StatusSNSResponse
	var err error
	var power float64

	if err = d.ExecCmd("Status 8", &res); err == nil {
		switch v := res.StatusSNS.Energy.Power.(type) {
		case float64:
			power = res.StatusSNS.Energy.Power.(float64)
		case []interface{}:
			// take first power meter value in case of a power meter list
			for i, vl := range v {
				switch vv := vl.(type) {
				case float64:
					if i == 0 {
						power = vv
					}
				}
			}
		}
	}

	return power, err
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
