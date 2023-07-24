package homewizard

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Connection is the homewizard connection
type Connection struct {
	*request.Helper
	uri     string
	channel int
}

// NewConnection creates a homewizard connection
func NewConnection(uri string) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	log := util.NewLogger("homewizard")
	c := &Connection{
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimRight(uri, "/"), "http"),
	}

	c.Client.Transport = request.NewTripper(log, transport.Insecure())

	return c, nil
}

// ExecCmd executes an api command and provides the response
func (d *Connection) ExecCmd(cmd string, res interface{}) error {
	return d.GetJSON(fmt.Sprintf("%s/api/v1/%s", d.uri, cmd), res)
}

// CurrentPower implements the api.Meter interface
func (d *Connection) CurrentPower() (float64, error) {
	var res DataResponse
	err := d.ExecCmd("data", &res)
	return res.ActivePowerW, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (d *Connection) TotalEnergy() (float64, error) {
	var res DataResponse
	err := d.ExecCmd("data", &res)
	return res.TotalPowerImportT1kWh, err
}
