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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// EVECUBE charger implementation
type EVECUBE struct {
	*request.Helper
	uri          string
	connector    int
	user, pass   string
	statusCache  time.Duration
	cache        time.Duration
	currentLimit int64 // stores the last non-zero current limit
}

func init() {
	registry.Add("evecube", NewEVECUBEFromConfig)
}

// EVECUBEStatus is the /api/status response
type EVECUBEStatus struct {
	ID                int       `json:"id"`
	Status            string    `json:"status"`
	Voltage           float64   `json:"voltage"`
	Current           float64   `json:"current"`
	MaxCurrent        int       `json:"maxCurrent"`
	Energy            int       `json:"energy"`      // Wh charged since transaction start
	EnergyTotal       float64   `json:"energyTotal"` // Total kWh
	LastSessionStart  time.Time `json:"lastSessionStart"`
	OCPPTransactionID int       `json:"ocppTransactionId"`
	CarConnected      bool      `json:"carConnected"`
	AuthTag           string    `json:"authenticationTag"`
	PhasesCurrent     []float64 `json:"phasesCurrent"` // Current in A on each phase [L1, L2, L3]
}

// EVECUBEUnitConfigRequest is the request body for /api/admin/unitconfig POST
type EVECUBEUnitConfigRequest struct {
	Values map[string]interface{} `json:"values"`
}

// NewEVECUBEFromConfig creates a EVECUBE charger from generic config
func NewEVECUBEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI       string
		User      string
		Password  string
		Connector int
		Cache     time.Duration
	}{
		Connector: 1,
		Cache:     time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Connector < 1 {
		return nil, fmt.Errorf("connector must be >= 1")
	}

	return NewEVECUBE(cc.URI, cc.User, cc.Password, cc.Connector, cc.Cache)
}

// NewEVECUBE creates EVECUBE charger
func NewEVECUBE(uri, user, password string, connector int, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("evecube")

	wb := &EVECUBE{
		Helper:       request.NewHelper(log),
		uri:          strings.TrimRight(uri, "/"),
		connector:    connector,
		user:         user,
		pass:         password,
		statusCache:  cache,
		cache:        cache,
		currentLimit: 6, // default to 6A
	}

	// Set basic auth if credentials provided
	if user != "" {
		wb.Client.Transport = transport.BasicAuth(user, password, wb.Client.Transport)
	}

	return wb, nil
}

func (wb *EVECUBE) getStatus() (EVECUBEStatus, error) {
	var statuses []EVECUBEStatus
	uri := fmt.Sprintf("%s/api/status", wb.uri)

	if err := wb.GetJSON(uri, &statuses); err != nil {
		return EVECUBEStatus{}, err
	}

	// Find the status for our connector
	for _, status := range statuses {
		if status.ID == wb.connector {
			return status, nil
		}
	}

	return EVECUBEStatus{}, fmt.Errorf("connector %d not found", wb.connector)
}

// Status implements the api.Charger interface
func (wb *EVECUBE) Status() (api.ChargeStatus, error) {
	status, err := wb.getStatus()
	if err != nil {
		return api.StatusNone, err
	}

	switch status.Status {
	case "Available":
		return api.StatusA, nil
	case "Preparing", "SuspendedEVSE", "SuspendedEV", "Finishing":
		return api.StatusB, nil
	case "Charging":
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown status: %s", status.Status)
	}
}

// Enabled implements the api.Charger interface
func (wb *EVECUBE) Enabled() (bool, error) {
	status, err := wb.getStatus()
	if err != nil {
		return false, err
	}

	return status.MaxCurrent > 0, nil
}

// Enable implements the api.Charger interface
func (wb *EVECUBE) Enable(enable bool) error {
	var current int64
	if enable {
		current = wb.currentLimit
	}

	return wb.setCurrent(current)
}

// MaxCurrent implements the api.Charger interface
func (wb *EVECUBE) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("current must be >= 6A")
	}

	wb.currentLimit = current

	return wb.setCurrent(current)
}

// setCurrent sets the MaxCurrent_X value via admin API
func (wb *EVECUBE) setCurrent(current int64) error {
	configKey := fmt.Sprintf("MaxCurrent_%d", wb.connector)
	reqBody := EVECUBEUnitConfigRequest{
		Values: map[string]interface{}{
			configKey: current,
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("%s/api/admin/unitconfig", wb.uri)
	req, err := request.New(http.MethodPost, uri, strings.NewReader(string(data)), request.JSONEncoding)
	if err != nil {
		return err
	}

	// Set basic auth for admin endpoint
	if wb.user != "" {
		req.SetBasicAuth(wb.user, wb.pass)
	}

	var resp map[string]interface{}
	if err := wb.DoJSON(req, &resp); err != nil {
		return err
	}

	return nil
}

var _ api.Meter = (*EVECUBE)(nil)

// CurrentPower implements the api.Meter interface
func (wb *EVECUBE) CurrentPower() (float64, error) {
	status, err := wb.getStatus()
	if err != nil {
		return 0, err
	}

	return status.Voltage * status.Current, nil
}

var _ api.MeterEnergy = (*EVECUBE)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *EVECUBE) TotalEnergy() (float64, error) {
	status, err := wb.getStatus()
	if err != nil {
		return 0, err
	}

	// EnergyTotal is in kWh
	return status.EnergyTotal, nil
}

var _ api.ChargeRater = (*EVECUBE)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *EVECUBE) ChargedEnergy() (float64, error) {
	status, err := wb.getStatus()
	if err != nil {
		return 0, err
	}

	// Energy is in Wh, convert to kWh
	return float64(status.Energy) / 1000.0, nil
}

var _ api.PhaseCurrents = (*EVECUBE)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *EVECUBE) Currents() (float64, float64, float64, error) {
	status, err := wb.getStatus()
	if err != nil {
		return 0, 0, 0, err
	}

	// PhasesCurrent contains [L1, L2, L3]
	if len(status.PhasesCurrent) >= 3 {
		return status.PhasesCurrent[0], status.PhasesCurrent[1], status.PhasesCurrent[2], nil
	}

	// Fallback if phase currents not available
	return 0, 0, 0, nil
}

/*
var _ api.PhaseVoltages = (*EVECUBE)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *EVECUBE) Voltages() (float64, float64, float64, error) {
	status, err := wb.getStatus()
	if err != nil {
		return 0, 0, 0, err
	}

	// The API only provides a single voltage value, assume same for all phases
	return status.Voltage, status.Voltage, status.Voltage, nil
}
*/

var _ api.Identifier = (*EVECUBE)(nil)

// Identify implements the api.Identifier interface
func (wb *EVECUBE) Identify() (string, error) {
	status, err := wb.getStatus()
	if err != nil {
		return "", err
	}

	return status.AuthTag, nil
}
