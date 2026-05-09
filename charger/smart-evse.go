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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// SmartEVSE-3.5 REST API charger implementation
// https://github.com/dingo35/SmartEVSE-3.5/blob/master/docs/REST_API.md

// SmartEVSE3 charger implementation
type SmartEVSE3 struct {
	*request.Helper
	implement.Caps
	uri  string
	curr int64
	mode int
	apiG util.Cacheable[smartEvseRestSettings]
}

// smartEvseRestSettings represents the JSON returned by GET /settings
type smartEvseRestSettings struct {
	Mode         string `json:"mode"`
	ModeID       int    `json:"mode_id"`
	CarConnected bool   `json:"car_connected"`
	Evse         struct {
		Connected    bool   `json:"connected"`
		Access       int    `json:"access"`
		Mode         int    `json:"mode"`
		ChargeTimer  int    `json:"charge_timer"`
		State        string `json:"state"`
		StateID      int    `json:"state_id"`
		Error        string `json:"error"`
		ErrorID      int    `json:"error_id"`
		RFID         string `json:"rfid"`
		RFIDReader   string `json:"rfidreader"`
		RFIDLastRead string `json:"rfid_lastread"`
		NrOfPhases   int    `json:"nrofphases"`
	} `json:"evse"`
	Settings struct {
		ChargeCurrent   int    `json:"charge_current"`
		OverrideCurrent int    `json:"override_current"`
		CurrentMin      int    `json:"current_min"`
		CurrentMax      int    `json:"current_max"`
		CurrentMain     int    `json:"current_main"`
		EnableC2        string `json:"enable_C2"`
	} `json:"settings"`
	EvMeter struct {
		Description       string  `json:"description"`
		Address           int     `json:"address"`
		ImportActivePower float64 `json:"import_active_power"`
		TotalWh           float64 `json:"total_wh"`
		ChargedWh         float64 `json:"charged_wh"`
		Currents          struct {
			Total float64 `json:"TOTAL"`
			L1    float64 `json:"L1"`
			L2    float64 `json:"L2"`
			L3    float64 `json:"L3"`
		} `json:"currents"`
		ImportActiveEnergy float64 `json:"import_active_energy"`
		ExportActiveEnergy float64 `json:"export_active_energy"`
	} `json:"ev_meter"`
}

// SmartEVSE-3.5 operating mode IDs
const (
	smartEvse3ModeOff    = 0
	smartEvse3ModeNormal = 1
	smartEvse3ModeSolar  = 2
	smartEvse3ModeSmart  = 3
	smartEvse3ModePause  = 4
)

// SmartEVSE-3.5 C2 contactor IDs
const (
	smartEvse3C2NotPresent = 0
	smartEvse3C2AlwaysOff  = 1 // single-phase
	smartEvse3C2SolarOff   = 2
	smartEvse3C2AlwaysOn   = 3 // three-phase
	smartEvse3C2Auto       = 4
)

func init() {
	registry.Add("smart-evse", NewSmartEVSE3FromConfig)
}

// NewSmartEVSE3FromConfig creates a SmartEVSE-3.5 REST charger from generic config
func NewSmartEVSE3FromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		URI        string
		Cache      time.Duration
		ChargeMode string
	}{
		Cache:      time.Second,
		ChargeMode: "normal",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, fmt.Errorf("missing uri")
	}

	mode := smartEvse3ModeNormal
	if cc.ChargeMode == "smart" {
		mode = smartEvse3ModeSmart
	}

	return NewSmartEVSE3(cc.URI, cc.Cache, mode)
}

// NewSmartEVSE3 creates a new SmartEVSE-3.5 REST charger
func NewSmartEVSE3(uri string, cache time.Duration, mode int) (api.Charger, error) {
	log := util.NewLogger("smart-evse")

	wb := &SmartEVSE3{
		Helper: request.NewHelper(log),
		Caps:   implement.New(),
		uri:    strings.TrimRight(util.DefaultScheme(uri, "http"), "/"),
		curr:   60, // 6 A in 1/10 A
		mode:   mode,
	}

	wb.Helper.Client.Transport = &http.Transport{
		DisableKeepAlives: true,
	}

	wb.apiG = util.ResettableCached(func() (smartEvseRestSettings, error) {
		var res smartEvseRestSettings
		err := wb.GetJSON(wb.uri+"/settings", &res)
		return res, err
	}, cache)

	// verify connectivity
	res, err := wb.apiG.Get()
	if err != nil {
		return nil, err
	}

	// decorate optional EV meter if configured in SmartEVSE
	if res.EvMeter.Description != "" && res.EvMeter.Description != "Disabled" {
		implement.Has(wb, implement.Meter(wb.currentPower))
		implement.Has(wb, implement.MeterImport(wb.totalEnergy))
		implement.Has(wb, implement.PhaseCurrents(wb.currents))
	}

	// decorate optional 1P/3P phase switching via C2 contactor
	if res.Settings.EnableC2 != "" && res.Settings.EnableC2 != "Not present" {
		implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3p))
		implement.Has(wb, implement.PhaseGetter(wb.getPhases))
	}

	// decorate optional RFID identification and status reason
	if res.Evse.RFIDReader != "" && res.Evse.RFIDReader != "Disabled" {
		implement.Has(wb, implement.Identifier(wb.identify))
		implement.Has(wb, implement.StatusReasoner(wb.statusReason))
	}

	return wb, nil
}

// post issues a POST request to the SmartEVSE with given query parameters.
// Mongoose webserver requires an empty request body to avoid timeout.
func (wb *SmartEVSE3) post(path string, query string) error {
	uri := fmt.Sprintf("%s/%s?%s", wb.uri, strings.TrimLeft(path, "/"), query)

	req, err := request.New(http.MethodPost, uri, strings.NewReader(""), request.URLEncoding)
	if err != nil {
		return err
	}

	_, err = wb.DoBody(req)
	wb.apiG.Reset()

	return err
}

func (wb *SmartEVSE3) setMode(mode int) error {
	return wb.post("settings", fmt.Sprintf("mode=%d", mode))
}

func (wb *SmartEVSE3) setOverrideCurrent(deciAmps int64) error {
	return wb.post("settings", fmt.Sprintf("override_current=%d", deciAmps))
}

// Status implements the api.Charger interface
func (wb *SmartEVSE3) Status() (api.ChargeStatus, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	if !res.CarConnected {
		return api.StatusA, nil
	}

	// state IDs from SmartEVSE firmware (main_c.h)
	switch res.Evse.StateID {
	case 2, 3, 6, 7, 10:
		// STATE_C, STATE_D, STATE_COMM_C, STATE_COMM_C_OK, STATE_C1
		return api.StatusC, nil
	default:
		return api.StatusB, nil
	}
}

// Enabled implements the api.Charger interface
func (wb *SmartEVSE3) Enabled() (bool, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return false, err
	}

	// STATE_B1 / STATE_C1: EVSE not ready, no PWM signal
	if res.Evse.StateID == 9 || res.Evse.StateID == 10 {
		return false, nil
	}

	if res.ModeID == smartEvse3ModeOff || res.ModeID == smartEvse3ModePause {
		return false, nil
	}

	return true, nil
}

// Enable implements the api.Charger interface
func (wb *SmartEVSE3) Enable(enable bool) error {
	res, err := wb.apiG.Get()
	if err != nil {
		return err
	}

	if enable {
		if res.ModeID != wb.mode {
			return wb.setMode(wb.mode)
		}
		return nil
	}

	if res.ModeID != smartEvse3ModeOff && res.ModeID != smartEvse3ModePause {
		return wb.setMode(smartEvse3ModePause)
	}

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *SmartEVSE3) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	deciAmps := current * 10

	if err := wb.setOverrideCurrent(deciAmps); err != nil {
		return err
	}

	wb.curr = deciAmps
	return nil
}

var _ api.ChargerEx = (*SmartEVSE3)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *SmartEVSE3) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	deciAmps := int64(current * 10)

	if err := wb.setOverrideCurrent(deciAmps); err != nil {
		return err
	}

	wb.curr = deciAmps
	return nil
}

// currentPower implements the api.Meter interface
func (wb *SmartEVSE3) currentPower() (float64, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return 0, err
	}

	// import_active_power is reported in W
	return res.EvMeter.ImportActivePower, nil
}

// totalEnergy implements the api.MeterImport interface
func (wb *SmartEVSE3) totalEnergy() (float64, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return 0, err
	}

	// import_active_energy is reported in Wh
	return res.EvMeter.ImportActiveEnergy / 1e3, nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *SmartEVSE3) phases1p3p(phases int) error {
	var c2 int
	switch phases {
	case 1:
		c2 = smartEvse3C2AlwaysOff
	case 3:
		c2 = smartEvse3C2AlwaysOn
	default:
		return fmt.Errorf("invalid phases: %d", phases)
	}

	// C2 switching takes effect on next state change; disable charger during switch
	return whenDisabled(wb, func() error {
		return wb.post("settings", fmt.Sprintf("enable_C2=%d", c2))
	})
}

// getPhases implements the api.PhaseGetter interface
func (wb *SmartEVSE3) getPhases() (int, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return 0, err
	}

	return res.Evse.NrOfPhases, nil
}

// currents implements the api.PhaseCurrents interface
func (wb *SmartEVSE3) currents() (float64, float64, float64, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	// phase currents are reported in 1/10 A
	return res.EvMeter.Currents.L1 / 10, res.EvMeter.Currents.L2 / 10, res.EvMeter.Currents.L3 / 10, nil
}

// statusReason implements the api.StatusReasoner interface
func (wb *SmartEVSE3) statusReason() (api.Reason, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return api.ReasonUnknown, err
	}

	// RFID reader enabled, car connected, but access not yet granted
	if res.CarConnected && res.Evse.Access == 0 {
		return api.ReasonWaitingForAuthorization, nil
	}

	return api.ReasonUnknown, nil
}

// identify implements the api.Identifier interface
func (wb *SmartEVSE3) identify() (string, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return "", err
	}

	return res.Evse.RFIDLastRead, nil
}

var _ api.Diagnosis = (*SmartEVSE3)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *SmartEVSE3) Diagnose() {
	res, err := wb.apiG.Get()
	if err != nil {
		fmt.Printf("\tError: %v\n", err)
		return
	}

	fmt.Printf("\tMode: %s (%d)\n", res.Mode, res.ModeID)
	fmt.Printf("\tCar connected: %t\n", res.CarConnected)

	fmt.Printf("\tEVSE connected: %t\n", res.Evse.Connected)
	fmt.Printf("\tEVSE access: %d\n", res.Evse.Access)
	fmt.Printf("\tEVSE mode: %d\n", res.Evse.Mode)
	fmt.Printf("\tEVSE charge timer: %d\n", res.Evse.ChargeTimer)
	fmt.Printf("\tEVSE state: %s (%d)\n", res.Evse.State, res.Evse.StateID)
	fmt.Printf("\tEVSE error: %s (%d)\n", res.Evse.Error, res.Evse.ErrorID)
	fmt.Printf("\tEVSE RFID: %s\n", res.Evse.RFID)
	fmt.Printf("\tEVSE RFID reader: %s\n", res.Evse.RFIDReader)
	fmt.Printf("\tEVSE RFID last read: %s\n", res.Evse.RFIDLastRead)
	fmt.Printf("\tEVSE nr of phases: %d\n", res.Evse.NrOfPhases)

	fmt.Printf("\tCharge current: %d A\n", res.Settings.ChargeCurrent)
	fmt.Printf("\tOverride current: %.1f A\n", float64(res.Settings.OverrideCurrent)/10)
	fmt.Printf("\tCurrent min/max: %d/%d A\n", res.Settings.CurrentMin, res.Settings.CurrentMax)
	fmt.Printf("\tCurrent main: %d A\n", res.Settings.CurrentMain)
	fmt.Printf("\tEnable C2: %s\n", res.Settings.EnableC2)
}
