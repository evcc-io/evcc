package homewizard

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Connection is the homewizard connection
type Connection struct {
	*request.Helper
	URI         string
	ProductType string
	Cache       time.Duration
	dataCache   provider.Cacheable[DataResponse]
	stateCache  provider.Cacheable[StateResponse]
}

// NewConnection creates a homewizard connection
func NewConnection(uri string, cache time.Duration) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	log := util.NewLogger("homewizard")
	c := &Connection{
		Helper: request.NewHelper(log),
		URI:    fmt.Sprintf("%s/api", util.DefaultScheme(strings.TrimRight(uri, "/"), "http")),
		Cache:  cache,
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

	c.dataCache = provider.ResettableCached(func() (DataResponse, error) {
		var res DataResponse
		err := c.GetJSON(fmt.Sprintf("%s/data", c.URI), &res)
		return res, err
	}, c.Cache)

	c.stateCache = provider.ResettableCached(func() (StateResponse, error) {
		var res StateResponse
		err := c.GetJSON(fmt.Sprintf("%s/state", c.URI), &res)
		return res, err
	}, c.Cache)

	return c, nil
}

// Enable implements the api.Charger interface
func (c *Connection) Enable(enable bool) error {
	var res StateResponse
	data := map[string]interface{}{
		"power_on": enable,
	}

	req, err := request.New(http.MethodPut, fmt.Sprintf("%s/state", c.URI), request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}
	if err := c.DoJSON(req, &res); err != nil {
		return err
	}

	if err == nil {
		c.stateCache.Reset()
		c.dataCache.Reset()
	}

	switch {
	case enable && !res.PowerOn:
		return errors.New("switchOn failed")
	case !enable && res.PowerOn:
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

// Enabled reads the homewizard switch state true=on/false=off
func (c *Connection) Enabled() (bool, error) {
	res, err := c.stateCache.Get()
	if err != nil {
		return false, err
	}
	return res.PowerOn, err
}

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	res, err := c.dataCache.Get()
	if err != nil {
		return 0, err
	}
	return res.ActivePowerW, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (c *Connection) TotalEnergy() (float64, error) {
	res, err := c.dataCache.Get()
	if err != nil {
		return 0, err
	}
	return res.TotalPowerImportT1kWh + res.TotalPowerImportT2kWh + res.TotalPowerImportT3kWh + res.TotalPowerImportT4kWh, err
}
