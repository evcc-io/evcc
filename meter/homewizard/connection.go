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
	URI         string
	ProductType string
}

// NewConnection creates a homewizard connection
func NewConnection(uri string) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	log := util.NewLogger("homewizard")
	c := &Connection{
		Helper: request.NewHelper(log),
		URI:    fmt.Sprintf("%s/api", util.DefaultScheme(strings.TrimRight(uri, "/"), "http")),
	}

	c.Client.Transport = request.NewTripper(log, transport.Insecure())

	// Check and set API version + product type
	var res ApiResponse
	if err := c.GetJSON(c.URI, &res); err != nil {
		return c, err
	}
	if res.ApiVersion != "v1" {
		return nil, errors.New("not supported api version: " + res.ApiVersion)
	}

	c.URI = c.URI + "/" + res.ApiVersion
	c.ProductType = res.ProductType

	return c, nil
}

// CurrentPower implements the api.Meter interface
func (d *Connection) CurrentPower() (float64, error) {
	var res DataResponse
	err := d.GetJSON(fmt.Sprintf("%s/data", d.URI), &res)
	return res.ActivePowerW, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (d *Connection) TotalEnergy() (float64, error) {
	var res DataResponse
	err := d.GetJSON(fmt.Sprintf("%s/data", d.URI), &res)
	return res.TotalPowerImportT1kWh + res.TotalPowerImportT2kWh + res.TotalPowerImportT3kWh + res.TotalPowerImportT4kWh, err
}
