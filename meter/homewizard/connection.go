package homewizard

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Connection is the homewizard connection
type Connection struct {
	*request.Helper
	uri string
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
func (d *Connection) ExecCmd(method, endpoint string, on bool, res interface{}) error {
	if method == "Get" {
		return d.GetJSON(fmt.Sprintf("%s/api/v1/%s", d.uri, endpoint), res)
	}
	if method == "Put" {
		data := map[string]interface{}{
			"power_on": on,
		}
		req, err := request.New(http.MethodPut, fmt.Sprintf("%s/api/v1/%s", d.uri, endpoint), request.MarshalJSON(data), request.JSONEncoding)
		if err != nil {
			return err
		}
		return d.DoJSON(req, &res)
	}
	return nil
}

// CurrentPower implements the api.Meter interface
func (d *Connection) CurrentPower() (float64, error) {
	var res DataResponse
	err := d.ExecCmd("Get", "data", &res)
	return res.ActivePowerW, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (d *Connection) TotalEnergy() (float64, error) {
	var res DataResponse
	err := d.ExecCmd("Get", "data", &res)
	return res.TotalPowerImportT1kWh, err
}
