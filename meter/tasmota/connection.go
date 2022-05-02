package tasmota

import (
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

// NewConnection creates Tasmota charger
func NewConnection(uri, user, password string) (*Connection, error) {
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

// CreateCmd creates the Tasmota command web request
// https://tasmota.github.io/docs/Commands/#with-web-requests
func (d *Connection) CreateCmd(cmd string) string {
	parameters := url.Values{
		"user":     []string{d.user},
		"password": []string{d.password},
		"cmnd":     []string{cmd},
	}

	return fmt.Sprintf("%s/cm?%s", d.uri, parameters.Encode())
}

// CurrentPower provides current power consumption
func (d *Connection) CurrentPower() (float64, error) {
	var resp StatusSNSResponse
	err := d.GetJSON(d.CreateCmd("Status 8"), &resp)
	if err != nil {
		return 0, err
	}

	return float64(resp.StatusSNS.Energy.Power), nil
}

// TotalEnergy implements the api.MeterEnergy interface
func (d *Connection) TotalEnergy() (float64, error) {
	var resp StatusSNSResponse
	err := d.GetJSON(d.CreateCmd("Status 8"), &resp)
	if err != nil {
		return 0, err
	}

	return float64(resp.StatusSNS.Energy.Total), nil
}
