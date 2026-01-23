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
)

// Connection is the homewizard connection
type Connection struct {
	*request.Helper
	uri         string
	usage       string
	phase       int
	ProductType string
	dataG       util.Cacheable[DataResponse]
	stateG      util.Cacheable[StateResponse]
}

// NewConnection creates a homewizard connection
func NewConnection(uri string, usage string, phase int, cache time.Duration) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	if phase < 1 || phase > 3 {
		return nil, errors.New("phase must be between 1 and 3")
	}

	log := util.NewLogger("homewizard")
	c := &Connection{
		Helper: request.NewHelper(log),
		uri:    fmt.Sprintf("%s/api", util.DefaultScheme(strings.TrimRight(uri, "/"), "http")),
		usage:  usage,
		phase:  phase,
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
	if c.usage == "pv" || c.usage == "battery" {
		return -res.ActivePowerW, err
	}
	return res.ActivePowerW, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (c *Connection) TotalEnergy() (float64, error) {
	res, err := c.dataG.Get()
	if c.usage == "pv" || c.usage == "battery" {
		return res.TotalPowerExportkWh, err
	}
	return res.TotalPowerImportkWh, err
}

// Currents implements the api.PhaseCurrents interface
func (c *Connection) Currents() (float64, float64, float64, error) {
	res, err := c.dataG.Get()

	// Single-phase meters only have one current reading
	if c.ProductType == "HWE-KWH1" || c.ProductType == "SDM230-wifi" {
		current := res.ActiveCurrentA
		if c.usage == "pv" || c.usage == "battery" {
			current = -current
		}

		// Return current on configured phase
		switch c.phase {
		case 1:
			return current, 0, 0, err
		case 2:
			return 0, current, 0, err
		case 3:
			return 0, 0, current, err
		}
	}

	// Three-phase meters have separate current readings per phase
	if c.usage == "pv" || c.usage == "battery" {
		return -res.ActiveCurrentL1A, -res.ActiveCurrentL2A, -res.ActiveCurrentL3A, err
	}
	return res.ActiveCurrentL1A, res.ActiveCurrentL2A, res.ActiveCurrentL3A, err
}

// Voltages implements the api.PhaseVoltages interface
func (c *Connection) Voltages() (float64, float64, float64, error) {
	res, err := c.dataG.Get()

	// Single-phase meters only have one voltage reading
	if c.ProductType == "HWE-KWH1" || c.ProductType == "SDM230-wifi" {
		voltage := res.ActiveVoltageV

		// Return voltage on configured phase
		switch c.phase {
		case 1:
			return voltage, 0, 0, err
		case 2:
			return 0, voltage, 0, err
		case 3:
			return 0, 0, voltage, err
		}
	}

	// Three-phase meters have separate voltage readings per phase
	return res.ActiveVoltageL1V, res.ActiveVoltageL2V, res.ActiveVoltageL3V, err
}
