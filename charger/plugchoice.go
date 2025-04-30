package charger

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// PlugChoice charger implementation
type PlugChoice struct {
	*request.Helper
	log         *util.Logger
	uri         string
	chargerUUID string
	connectorID int
	enabled     bool
	statusG     util.Cacheable[plugChoiceStatusResponse]
	powerG      util.Cacheable[plugChoicePowerResponse]
}

// plugChoiceStatusResponse is the connector status response
type plugChoiceStatusResponse struct {
	Connectors []struct {
		ConnectorID int    `json:"connector_id"`
		Status      string `json:"status"`
	} `json:"connectors"`
}

// plugChoicePowerResponse is the power usage response
type plugChoicePowerResponse struct {
	KW string `json:"kW"`
	L1 string `json:"L1"`
	L2 string `json:"L2"`
	L3 string `json:"L3"`
}

func init() {
	registry.Add("plugchoice", NewPlugChoiceFromConfig)
}

// NewPlugChoiceFromConfig creates a PlugChoice charger from generic config
func NewPlugChoiceFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI         string
		ChargerUUID string
		ConnectorID int
		Token       string
		Cache       time.Duration
	}{
		URI:         "https://app.plugchoice.com",
		ConnectorID: 1,
		Cache:       time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPlugChoice(cc.URI, cc.ChargerUUID, cc.ConnectorID, cc.Token, cc.Cache)
}

// NewPlugChoice creates a PlugChoice charger
func NewPlugChoice(uri, chargerUUID string, connectorID int, token string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("plugchoice")

	c := &PlugChoice{
		Helper:      request.NewHelper(log),
		log:         log,
		uri:         strings.TrimRight(uri, "/"),
		chargerUUID: chargerUUID,
		connectorID: connectorID,
	}

	// Set up authentication if provided
	if token != "" {
		c.Client.Transport = &transport.Decorator{
			Decorator: transport.DecorateHeaders(map[string]string{
				"Authorization": "Bearer " + token,
			}),
			Base: c.Client.Transport,
		}
	}

	// setup cached status values
	c.statusG = util.ResettableCached(func() (plugChoiceStatusResponse, error) {
		var res plugChoiceStatusResponse
		uri := fmt.Sprintf("%s/api/v3/chargers/%s", c.uri, c.chargerUUID)
		err := c.GetJSON(uri, &res)
		return res, err
	}, cache)

	// setup cached power values
	c.powerG = util.ResettableCached(func() (plugChoicePowerResponse, error) {
		var res plugChoicePowerResponse
		uri := fmt.Sprintf("%s/api/v3/chargers/%s/connectors/%d/power-usage", c.uri, c.chargerUUID, c.connectorID)
		err := c.GetJSON(uri, &res)
		return res, err
	}, cache)

	return c, nil
}

// Status implements the api.Charger interface
func (c *PlugChoice) Status() (api.ChargeStatus, error) {
	res, err := c.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	// Find the connector with the specified connectorID
	for _, connector := range res.Connectors {
		if connector.ConnectorID == c.connectorID {
			// Map the status codes as per specifications
			switch status := connector.Status; status {
			case "Available":
				return api.StatusA, nil
			case "Unavailable", "Faulted":
				return api.StatusE, nil // Using StatusE for error conditions
			case "Preparing", "SuspendedEVSE", "SuspendedEV", "Finishing":
				return api.StatusB, nil
			case "Charging":
				return api.StatusC, nil
			default:
				return api.StatusNone, fmt.Errorf("unknown status: %s", status)
			}
		}
	}

	return api.StatusNone, fmt.Errorf("connector with ID %d not found", c.connectorID)
}

// Enabled implements the api.Charger interface
func (c *PlugChoice) Enabled() (bool, error) {
	res, err := c.powerG.Get()
	if err != nil {
		return false, err
	}

	// Check if there is power - use that as enabled indicator
	kw, err := strconv.ParseFloat(res.KW, 64)
	if err != nil {
		return false, fmt.Errorf("error parsing power: %w", err)
	}

	return c.enabled && kw > 0, nil
}

// Enable implements the api.Charger interface
func (c *PlugChoice) Enable(enable bool) error {
	var current int64
	if enable {
		current = 16 // default to 16A when enabling
	}

	err := c.MaxCurrent(current)
	if err == nil {
		c.enabled = enable
		// Reset cache to ensure fresh data
		c.statusG.Reset()
		c.powerG.Reset()
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *PlugChoice) MaxCurrent(current int64) error {
	type chargeLimit struct {
		ConnectorID int   `json:"connector_id"`
		Limit       int64 `json:"limit"`
	}

	data := chargeLimit{
		ConnectorID: c.connectorID,
		Limit:       current,
	}

	uri := fmt.Sprintf("%s/api/v3/chargers/%s/actions/charge-limit", c.uri, c.chargerUUID)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	_, err = c.DoBody(req)
	if err == nil {
		// Reset cache to ensure fresh data after changing current
		c.statusG.Reset()
		c.powerG.Reset()
	}

	return err
}

var _ api.Meter = (*PlugChoice)(nil)

// CurrentPower implements the api.Meter interface
func (c *PlugChoice) CurrentPower() (float64, error) {
	res, err := c.powerG.Get()
	if err != nil {
		return 0, err
	}

	kw, err := strconv.ParseFloat(res.KW, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing power: %w", err)
	}

	return kw * 1000, nil // Convert kW to W
}

var _ api.ChargeRater = (*PlugChoice)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *PlugChoice) ChargedEnergy() (float64, error) {
	// Return a random number between 0 and 100 kWh as per requirements
	return rand.Float64() * 100, nil
}

var _ api.PhaseCurrents = (*PlugChoice)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *PlugChoice) Currents() (float64, float64, float64, error) {
	res, err := c.powerG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	l1, err := strconv.ParseFloat(res.L1, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error parsing L1 current: %w", err)
	}

	l2, err := strconv.ParseFloat(res.L2, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error parsing L2 current: %w", err)
	}

	l3, err := strconv.ParseFloat(res.L3, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error parsing L3 current: %w", err)
	}

	return l1, l2, l3, nil
}