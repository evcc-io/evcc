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
	"github.com/evcc-io/evcc/util/sponsor"
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
	currentLimit int64
}

func init() {
	registry.Add("evecube", NewEVECUBEFromConfig)
}

//go:generate go tool decorate -f decorateEVECUBE -b *EVECUBE -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.Identifier,Identify,func() (string, error)"

// EVECUBEUnitConfig is the /api/admin/unitconfig response
type EVECUBEUnitConfig struct {
	ForcePhaseCharging int `json:"ForcePhaseCharging"`
	NumberOfConnectors int `json:"NumberOfConnectors"`
	MaxCurrent1        int `json:"MaxCurrent_1"`
	MaxCurrent2        int `json:"MaxCurrent_2"`
	MaxCurrent3        int `json:"MaxCurrent_3"`
	MaxCurrent4        int `json:"MaxCurrent_4"`
}

// EVECUBEStatusResponse is the /api/admin/status response
type EVECUBEStatusResponse struct {
	Connectors []EVECUBEConnectorStatus `json:"connectors"`
}

// EVECUBEConnectorStatus represents a single connector in the admin status response
type EVECUBEConnectorStatus struct {
	ID               int                     `json:"id"`
	Status           string                  `json:"status"`
	Voltage          float64                 `json:"voltage"`
	Voltages         []float64               `json:"voltages"`
	Current          float64                 `json:"current"`
	Currents         []float64               `json:"currents"`
	Energy           float64                 `json:"energy"`
	EnergyTotal      float64                 `json:"energyTotal"`
	CarConnected     bool                    `json:"carConnected"`
	LastStatusPacket EVECUBELastStatusPacket `json:"lastStatusPacket"`
}

// EVECUBELastStatusPacket contains detailed status information
type EVECUBELastStatusPacket struct {
	CarStatus      string    `json:"carStatus"`
	ChargingStatus string    `json:"chargingStatus"`
	Voltage        float64   `json:"voltage"`
	Voltages       []float64 `json:"voltages"`
	Current        float64   `json:"current"`
	ActualWh       float64   `json:"actualWh"`
	TotalWh        float64   `json:"totalWh"`
}

// EVECUBEAutomationStatus is the /api/admin/automation/status response
type EVECUBEAutomationStatus struct {
	Connectors map[string]any `json:"connectors"`
	AuthTag    struct {
		Tag string `json:"tag"`
	} `json:"authTag"`
}

// EVECUBEUnitConfigRequest is the request body for /api/admin/unitconfig POST
type EVECUBEUnitConfigRequest struct {
	Values map[string]any `json:"values"`
}

// NewEVECUBEFromConfig creates a EVECUBE charger from generic config
func NewEVECUBEFromConfig(other map[string]any) (api.Charger, error) {
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

	wb, err := NewEVECUBE(cc.URI, cc.User, cc.Password, cc.Connector, cc.Cache)
	if err != nil {
		return nil, err
	}

	// Get unit configuration to determine connector count
	config, err := wb.getUnitConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get unit config: %w", err)
	}

	if cc.Connector < 1 || cc.Connector > 4 || config.NumberOfConnectors < cc.Connector {
		return nil, fmt.Errorf("invalid connector: %d", cc.Connector)
	}

	// Phases1p3p and Identify APIs affect the entire charger, not individual connectors
	// Only enable these APIs if the charger has a single connector
	var phases1p3p func(int) error
	var identify func() (string, error)

	if config.NumberOfConnectors == 1 {
		phases1p3p = wb.phases1p3p
		identify = wb.identify
	}

	return decorateEVECUBE(wb, phases1p3p, identify), nil
}

// NewEVECUBE creates EVECUBE charger
func NewEVECUBE(uri, user, password string, connector int, cache time.Duration) (*EVECUBE, error) {
	log := util.NewLogger("evecube")

	wb := &EVECUBE{
		Helper:       request.NewHelper(log),
		uri:          strings.TrimRight(uri, "/"),
		connector:    connector,
		user:         user,
		pass:         password,
		statusCache:  cache,
		cache:        cache,
		currentLimit: 6,
	}

	if user != "" {
		wb.Client.Transport = transport.BasicAuth(user, password, wb.Client.Transport)
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	return wb, nil
}

func (wb *EVECUBE) getUnitConfig() (EVECUBEUnitConfig, error) {
	var config EVECUBEUnitConfig
	uri := fmt.Sprintf("%s/api/admin/unitconfig", wb.uri)

	if err := wb.GetJSON(uri, &config); err != nil {
		return EVECUBEUnitConfig{}, err
	}

	return config, nil
}

func (wb *EVECUBE) getStatus() (EVECUBEConnectorStatus, error) {
	var resp EVECUBEStatusResponse
	uri := fmt.Sprintf("%s/api/admin/status", wb.uri)

	if err := wb.GetJSON(uri, &resp); err != nil {
		return EVECUBEConnectorStatus{}, err
	}

	for _, connector := range resp.Connectors {
		if connector.ID == wb.connector {
			return connector, nil
		}
	}

	return EVECUBEConnectorStatus{}, fmt.Errorf("connector %d not found", wb.connector)
}

func (wb *EVECUBE) getAutomationStatus() (EVECUBEAutomationStatus, error) {
	var resp EVECUBEAutomationStatus
	uri := fmt.Sprintf("%s/api/admin/automation/status", wb.uri)

	if err := wb.GetJSON(uri, &resp); err != nil {
		return EVECUBEAutomationStatus{}, err
	}

	return resp, nil
}

// Status implements the api.Charger interface
func (wb *EVECUBE) Status() (api.ChargeStatus, error) {
	status, err := wb.getStatus()
	if err != nil {
		return api.StatusNone, err
	}

	if !status.CarConnected {
		return api.StatusA, nil
	}

	if status.Status == "Charging" {
		return api.StatusC, nil
	}

	return api.StatusB, nil
}

// Enabled implements the api.Charger interface
func (wb *EVECUBE) Enabled() (bool, error) {
	config, err := wb.getUnitConfig()
	if err != nil {
		return false, err
	}

	// Get the MaxCurrent for our connector
	var maxCurrent int
	switch wb.connector {
	case 1:
		maxCurrent = config.MaxCurrent1
	case 2:
		maxCurrent = config.MaxCurrent2
	case 3:
		maxCurrent = config.MaxCurrent3
	case 4:
		maxCurrent = config.MaxCurrent4
	}

	return maxCurrent > 0, nil
}

// Enable implements the api.Charger interface
func (wb *EVECUBE) Enable(enable bool) error {
	var current int64
	if enable {
		current = wb.currentLimit
	}

	return wb.setValue(fmt.Sprintf("MaxCurrent_%d", wb.connector), current)
}

// MaxCurrent implements the api.Charger interface
func (wb *EVECUBE) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("current must be >= 6A")
	}

	wb.currentLimit = current

	return wb.setValue(fmt.Sprintf("MaxCurrent_%d", wb.connector), current)
}

// setValue sets a named value via admin API
func (wb *EVECUBE) setValue(key string, value int64) error {
	reqBody := EVECUBEUnitConfigRequest{
		Values: map[string]any{
			key: value,
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

	if wb.user != "" {
		req.SetBasicAuth(wb.user, wb.pass)
	}

	var resp map[string]any
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

	return status.EnergyTotal, nil
}

var _ api.ChargeRater = (*EVECUBE)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *EVECUBE) ChargedEnergy() (float64, error) {
	status, err := wb.getStatus()
	if err != nil {
		return 0, err
	}

	return float64(status.Energy) / 1000.0, nil
}

var _ api.PhaseCurrents = (*EVECUBE)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *EVECUBE) Currents() (float64, float64, float64, error) {
	status, err := wb.getStatus()
	if err != nil {
		return 0, 0, 0, err
	}

	if len(status.Currents) == 3 {
		return status.Currents[0], status.Currents[1], status.Currents[2], nil
	}

	return 0, 0, 0, nil
}

var _ api.PhaseVoltages = (*EVECUBE)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *EVECUBE) Voltages() (float64, float64, float64, error) {
	status, err := wb.getStatus()
	if err != nil {
		return 0, 0, 0, err
	}

	if len(status.LastStatusPacket.Voltages) == 3 {
		return status.LastStatusPacket.Voltages[0], status.LastStatusPacket.Voltages[1], status.LastStatusPacket.Voltages[2], nil
	}

	return 0, 0, 0, nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *EVECUBE) phases1p3p(phases int) error {
	return whenDisabled(wb, func() error {
		return wb.setValue("ForcePhaseCharging", int64(phases))
	})
}

// identify implements the api.Identifier interface
func (wb *EVECUBE) identify() (string, error) {
	status, err := wb.getAutomationStatus()
	if err != nil {
		return "", err
	}

	return status.AuthTag.Tag, nil
}
