package charger

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

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
	"errors"
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
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

// Plugchoice charger implementation
type Plugchoice struct {
	*request.Helper
	uri       string
	uuid      string
	connector int
	enabled   bool
	current   int64
	statusG   util.Cacheable[plugchoice.StatusResponse]
	powerG    util.Cacheable[plugchoice.PowerResponse]
}

func init() {
	registry.Add("plugchoice", NewPlugchoiceFromConfig)
}

// NewPlugchoiceFromConfig creates a Plugchoice charger from generic config
func NewPlugchoiceFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		URI       string
		UUID      string // kept for backward compatibility
		Identity  string
		Connector int
		Token     string
		Cache     time.Duration
	}{
		URI:       "https://app.plugchoice.com",
		Connector: 1,
		Cache:     time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPlugchoice(cc.URI, cc.UUID, cc.Identity, cc.Connector, cc.Token, cc.Cache)
}

// NewPlugchoice creates a Plugchoice charger
func NewPlugchoice(uri, uuid, identity string, connector int, token string, cache time.Duration) (api.Charger, error) {
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

	// If both are provided, Identity takes precedence
	if identity == "" && uuid == "" {
		return nil, errors.New("either identity or uuid are required")
	}

	// If identity is provided but no UUID, try to find the UUID
	if uuid == "" && identity != "" {
		var err error
		uuid, err = plugchoice.FindUUIDByIdentity(helper, uri, identity)
		if err != nil {
			return nil, fmt.Errorf("error finding charger UUID: %w", err)
		}
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	c := &Plugchoice{
		Helper:    helper,
		uri:       uri,
		uuid:      uuid,
		connector: connector,
		current:   6,
	}

	// setup cached status values
	c.statusG = util.ResettableCached(func() (plugchoice.StatusResponse, error) {
		var res plugchoice.StatusResponse
		uri := fmt.Sprintf("%s/api/v3/chargers/%s", c.uri, c.uuid)
		err := c.GetJSON(uri, &res)
		return res, err
	}, cache)

	// setup cached power values
	c.powerG = util.ResettableCached(func() (plugchoice.PowerResponse, error) {
		var res plugchoice.PowerResponse
		uri := fmt.Sprintf("%s/api/v3/chargers/%s/connectors/%d/power-usage", c.uri, c.uuid, c.connector)
		err := c.GetJSON(uri, &res)
		return res, err
	}, cache)

	return c, nil
}

// Status implements the api.Charger interface
func (c *Plugchoice) Status() (api.ChargeStatus, error) {
	res, err := c.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	// Find the connector with the specified connector
	for _, connector := range res.Data.Connectors {
		if connector.ConnectorID == c.connector {
			// Map the status codes as per specifications
			switch status := connector.Status; status {
			case core.ChargePointStatusAvailable:
				return api.StatusA, nil
			case core.ChargePointStatusUnavailable, core.ChargePointStatusFaulted:
				return api.StatusE, nil // Using StatusE for error conditions
			case core.ChargePointStatusPreparing, core.ChargePointStatusSuspendedEVSE, core.ChargePointStatusSuspendedEV, core.ChargePointStatusFinishing:
				return api.StatusB, nil
			case core.ChargePointStatusCharging:
				return api.StatusC, nil
			default:
				return api.StatusNone, fmt.Errorf("unknown status: %s", status)
			}
		}
	}

	return api.StatusNone, fmt.Errorf("connector with ID %d not found", c.connector)
}

// Enabled implements the api.Charger interface
func (c *Plugchoice) Enabled() (bool, error) {
	res, err := c.statusG.Get()
	if err != nil {
		return false, err
	}

	// Find the connector with the specified connector
	for _, connector := range res.Data.Connectors {
		if connector.ConnectorID == c.connector {
			// Check status for enabled state
			switch status := connector.Status; status {
			case core.ChargePointStatusCharging, core.ChargePointStatusSuspendedEV:
				return true, nil
			case core.ChargePointStatusSuspendedEVSE:
				return false, nil
			default:
				return c.enabled, nil
			}
		}
	}

	return false, fmt.Errorf("connector with ID %d not found", c.connector)
}

// Enable implements the api.Charger interface
func (c *Plugchoice) Enable(enable bool) error {
	var current int64
	if enable {
		current = c.current
	}

	err := c.maxCurrent(current)
	if err == nil {
		c.enabled = enable
	}

	return err
}

func (c *Plugchoice) maxCurrent(current int64) error {
	type chargeLimit struct {
		Connector int   `json:"connector_id"`
		Limit     int64 `json:"limit"`
	}

	data := chargeLimit{
		Connector: c.connector,
		Limit:     current,
	}

	uri := fmt.Sprintf("%s/api/v3/chargers/%s/actions/charge-limit", c.uri, c.uuid)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	_, err := c.Do(req)
	if err == nil {
		c.statusG.Reset()
		c.powerG.Reset()
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *Plugchoice) MaxCurrent(current int64) error {
	err := c.maxCurrent(current)
	if err == nil {
		c.current = current
	}

	return err
}

var _ api.Meter = (*Plugchoice)(nil)

// CurrentPower implements the api.Meter interface
func (c *Plugchoice) CurrentPower() (float64, error) {
	// Should be zero if not enabled
	if !c.enabled {
		return 0, nil
	}

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

var _ api.PhaseCurrents = (*Plugchoice)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Plugchoice) Currents() (float64, float64, float64, error) {
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
