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

// Product type constants
const (
	ProductTypeKWH1   = "HWE-KWH1"    // Single-phase kWh meter
	ProductTypeKWH3   = "HWE-KWH3"    // Three-phase kWh meter
	ProductTypeSDM230 = "SDM230-wifi" // Single-phase kWh meter
	ProductTypeSDM630 = "SDM630-wifi" // Three-phase kWh meter
	ProductTypeSocket = "HWE-SKT"     // Smart socket
	ProductTypeP1     = "HWE-P1"      // P1 meter
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

// isSinglePhase returns true if the product type is a single-phase meter
func isSinglePhase(productType string) bool {
	switch productType {
	case ProductTypeKWH1, ProductTypeSDM230:
		return true
	default:
		return false
	}
}

// mapValueToPhase maps a single-phase value to the specified phase (L1, L2, or L3)
func mapValueToPhase(value float64, phase int) (float64, float64, float64) {
	switch phase {
	case 1:
		return value, 0, 0
	case 2:
		return 0, value, 0
	case 3:
		return 0, 0, value
	default:
		return value, 0, 0 // fallback to L1
	}
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

// Powers implements the PhasePowers interface
func (c *Connection) Powers() (float64, float64, float64, error) {
	res, err := c.dataG.Get()

	if isSinglePhase(c.ProductType) {
		power := res.ActivePowerW
		if c.usage == "pv" || c.usage == "battery" {
			power = -power
		}

		l1, l2, l3 := mapValueToPhase(power, c.phase)
		return l1, l2, l3, err
	}

	if c.usage == "pv" || c.usage == "battery" {
		return -res.ActivePowerL1W, -res.ActivePowerL2W, -res.ActivePowerL3W, err
	}

	return res.ActivePowerL1W, res.ActivePowerL2W, res.ActivePowerL3W, err
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
	if isSinglePhase(c.ProductType) {
		current := res.ActiveCurrentA
		if c.usage == "pv" || c.usage == "battery" {
			current = -current
		}

		l1, l2, l3 := mapValueToPhase(current, c.phase)
		return l1, l2, l3, err
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
	if isSinglePhase(c.ProductType) {
		l1, l2, l3 := mapValueToPhase(res.ActiveVoltageV, c.phase)
		return l1, l2, l3, err
	}

	// Three-phase meters have separate voltage readings per phase
	return res.ActiveVoltageL1V, res.ActiveVoltageL2V, res.ActiveVoltageL3V, err
}
