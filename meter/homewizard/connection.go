package homewizard

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
)

// Connection is the homewizard connection
type Connection struct {
	*request.Helper
	uri         string
	usage       string
	ProductType string
	dataG       util.Cacheable[DataResponse]
	stateG      util.Cacheable[StateResponse]
}

// NewConnection creates a homewizard connection
func NewConnection(uri string, usage string, cache time.Duration) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	log := util.NewLogger("homewizard")
	c := &Connection{
		Helper: request.NewHelper(log),
		uri:    fmt.Sprintf("%s/api", util.DefaultScheme(strings.TrimRight(uri, "/"), "http")),
		usage:  usage,
	}

	c.Client.Transport = request.NewTripper(log, transport.Insecure())

	// check and set API version + product type
	var res ApiResponse
	if err := c.GetJSON(c.uri, &res); err != nil {
		return nil, err
	}
	if res.ApiVersion != "v1" {
		return nil, errors.New("unsupported api version: " + res.ApiVersion)
	}

	c.uri = c.uri + "/" + res.ApiVersion
	c.ProductType = res.ProductType

	c.dataG = util.ResettableCached(func() (DataResponse, error) {
		var res DataResponse
		err := c.GetJSON(fmt.Sprintf("%s/data", c.uri), &res)
		return res, err
	}, cache)

	c.stateG = util.ResettableCached(func() (StateResponse, error) {
		var res StateResponse
		err := c.GetJSON(fmt.Sprintf("%s/state", c.uri), &res)
		return res, err
	}, cache)

	return c, nil
}

// Enable implements the api.Charger interface
func (c *Connection) Enable(enable bool) error {
	var res StateResponse
	data := map[string]any{
		"power_on": enable,
	}

	req, err := request.New(http.MethodPut, fmt.Sprintf("%s/state", c.uri), request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}
	if err := c.DoJSON(req, &res); err != nil {
		return err
	}

	c.stateG.Reset()
	c.dataG.Reset()

	switch {
	case enable && !res.PowerOn:
		return errors.New("switchOn failed")
	case !enable && res.PowerOn:
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

// Enabled implements the api.Charger interface
func (c *Connection) Enabled() (bool, error) {
	res, err := c.stateG.Get()
	return res.PowerOn, err
}

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	res, err := c.dataG.Get()
	if c.usage == "pv" {
		return -res.ActivePowerW, err
	}
	return res.ActivePowerW, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (c *Connection) TotalEnergy() (float64, error) {
	res, err := c.dataG.Get()
	if c.usage == "pv" {
		return res.TotalPowerExportT1kWh + res.TotalPowerExportT2kWh + res.TotalPowerExportT3kWh + res.TotalPowerExportT4kWh, err
	}
	return res.TotalPowerImportT1kWh + res.TotalPowerImportT2kWh + res.TotalPowerImportT3kWh + res.TotalPowerImportT4kWh, err
}

// Currents implements the api.PhaseCurrents interface
func (c *Connection) Currents() (float64, float64, float64, error) {
	res, err := c.dataG.Get()

	// 1p meters report the total instead of per-phase values
	l1, l2, l3 := res.ActiveCurrentA, 0.0, 0.0
	if res.ActiveCurrentL1A != nil {
		l1, l2, l3 = lo.FromPtr(res.ActiveCurrentL1A), lo.FromPtr(res.ActiveCurrentL2A), lo.FromPtr(res.ActiveCurrentL3A)
	}

	if c.usage == "pv" {
		return -l1, -l2, -l3, err
	}
	return l1, l2, l3, err
}

// Powers implements the api.PhasePowers interface
func (c *Connection) Powers() (float64, float64, float64, error) {
	res, err := c.dataG.Get()
	if c.usage == "pv" {
		return -res.ActivePowerL1W, -res.ActivePowerL2W, -res.ActivePowerL3W, err
	}
	return res.ActivePowerL1W, res.ActivePowerL2W, res.ActivePowerL3W, err
}

// Voltages implements the api.PhaseVoltages interface
func (c *Connection) Voltages() (float64, float64, float64, error) {
	res, err := c.dataG.Get()

	// 1p meters report the total instead of per-phase values
	l1, l2, l3 := res.ActiveVoltageV, 0.0, 0.0
	if res.ActiveCurrentL1A != nil {
		l1, l2, l3 = lo.FromPtr(res.ActiveVoltageL1V), lo.FromPtr(res.ActiveVoltageL2V), lo.FromPtr(res.ActiveVoltageL3V)
	}

	return l1, l2, l3, err
}
