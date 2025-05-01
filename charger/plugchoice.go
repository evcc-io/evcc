package charger

// LICENSE

// Copyright (c) 2025 andig

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/plugchoice"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
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
	statusG     util.Cacheable[plugchoice.StatusResponse]
	powerG      util.Cacheable[plugchoice.PowerResponse]
}

func init() {
	registry.Add("plugchoice", NewPlugChoiceFromConfig)
}

// NewPlugChoiceFromConfig creates a PlugChoice charger from generic config
func NewPlugChoiceFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI         string
		ChargerUUID string // kept for backward compatibility
		Identity    string
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

	// If both are provided, Identity takes precedence
	if cc.Identity != "" || cc.ChargerUUID != "" {
		return NewPlugChoice(cc.URI, cc.ChargerUUID, cc.Identity, cc.ConnectorID, cc.Token, cc.Cache)
	}

	return nil, fmt.Errorf("either identity or chargerUUID must be provided")
}

// NewPlugChoice creates a PlugChoice charger
func NewPlugChoice(uri, chargerUUID, identity string, connectorID int, token string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("plugchoice")
	helper := request.NewHelper(log)
	uri = strings.TrimRight(uri, "/")

	// Set up authentication if provided
	if token != "" {
		helper.Client.Transport = &transport.Decorator{
			Decorator: transport.DecorateHeaders(map[string]string{
				"Authorization": "Bearer " + token,
			}),
			Base: helper.Client.Transport,
		}
	}

	// If identity is provided but no UUID, try to find the UUID
	if chargerUUID == "" && identity != "" {
		var err error
		chargerUUID, err = plugchoice.FindChargerUUIDByIdentity(log, helper, uri, identity)
		if err != nil {
			return nil, fmt.Errorf("error finding charger UUID: %w", err)
		}
	}

	// If we still don't have a UUID, return an error
	if chargerUUID == "" {
		return nil, fmt.Errorf("either chargerUUID or identity must be provided")
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	c := &PlugChoice{
		Helper:      helper,
		log:         log,
		uri:         uri,
		chargerUUID: chargerUUID,
		connectorID: connectorID,
	}

	// setup cached status values
	c.statusG = util.ResettableCached(func() (plugchoice.StatusResponse, error) {
		var res plugchoice.StatusResponse
		uri := fmt.Sprintf("%s/api/v3/chargers/%s", c.uri, c.chargerUUID)
		err := c.GetJSON(uri, &res)
		return res, err
	}, cache)

	// setup cached power values
	c.powerG = util.ResettableCached(func() (plugchoice.PowerResponse, error) {
		var res plugchoice.PowerResponse
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
	for _, connector := range res.Data.Connectors {
		if connector.ConnectorID == c.connectorID {
			// Map the status codes as per specifications
			switch status := connector.Status; status {
			case plugchoice.StatusAvailable:
				return api.StatusA, nil
			case plugchoice.StatusUnavailable, plugchoice.StatusFaulted:
				return api.StatusE, nil // Using StatusE for error conditions
			case plugchoice.StatusPreparing, plugchoice.StatusSuspendedEVSE, plugchoice.StatusSuspendedEV, plugchoice.StatusFinishing:
				return api.StatusB, nil
			case plugchoice.StatusCharging:
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
	res, err := c.statusG.Get()
	if err != nil {
		return false, err
	}

	// Find the connector with the specified connectorID
	for _, connector := range res.Data.Connectors {
		if connector.ConnectorID == c.connectorID {
			// Check status for enabled state
			switch status := connector.Status; status {
			case plugchoice.StatusCharging, plugchoice.StatusSuspendedEV:
				return true, nil
			case plugchoice.StatusSuspendedEVSE:
				return false, nil
			default:
				return c.enabled, nil
			}
		}
	}

	return false, fmt.Errorf("connector with ID %d not found", c.connectorID)
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

	// Handle the case where power value is "-"
	if res.KW == "-" {
		return 0, nil
	}

	kw, err := strconv.ParseFloat(res.KW, 64)
	if err != nil {
		return 0, err
	}

	return kw * 1000, nil // Convert kW to W
}

var _ api.PhaseCurrents = (*PlugChoice)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *PlugChoice) Currents() (float64, float64, float64, error) {
	res, err := c.powerG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	// Helper function to parse current values, handling "-" as 0
	parsePhaseValue := func(val string, phase string) (float64, error) {
		if val == "-" {
			return 0, nil
		}
		res, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("parsing %s current: %w", phase, err)
		}
		return res, nil
	}

	l1, err := parsePhaseValue(res.L1, "L1")
	if err != nil {
		return 0, 0, 0, err
	}

	l2, err := parsePhaseValue(res.L2, "L2")
	if err != nil {
		return 0, 0, 0, err
	}

	l3, err := parsePhaseValue(res.L3, "L3")
	if err != nil {
		return 0, 0, 0, err
	}

	return l1, l2, l3, nil
}
