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
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

// SmartEVSE-3.5 REST API charger implementation
// https://github.com/dingo35/SmartEVSE-3.5/blob/master/docs/REST_API.md

// SmartEVSE3 charger implementation
type SmartEVSE3 struct {
	*request.Helper
	uri  string
	curr int64
	apiG util.Cacheable[smartEvseRestSettings]
}

// smartEvseRestSettings represents the JSON returned by GET /settings
type smartEvseRestSettings struct {
	Mode         string `json:"mode"`
	ModeID       int    `json:"mode_id"`
	CarConnected bool   `json:"car_connected"`
	Evse         struct {
		Temp           int    `json:"temp"`
		Connected      int    `json:"connected"`
		Access         int    `json:"access"`
		Mode           int    `json:"mode"`
		ChargeTimer    int    `json:"charge_timer"`
		SolarStopTimer int    `json:"solar_stop_timer"`
		State          string `json:"state"`
		StateID        int    `json:"state_id"`
		Error          string `json:"error"`
		ErrorID        int    `json:"error_id"`
		RFID           string `json:"rfid"`
	} `json:"evse"`
	Settings struct {
		ChargeCurrent     int `json:"charge_current"`
		OverrideCurrent   int `json:"override_current"`
		CurrentMin        int `json:"current_min"`
		CurrentMax        int `json:"current_max"`
		CurrentMain       int `json:"current_main"`
		SolarMaxImport    int `json:"solar_max_import"`
		SolarStartCurrent int `json:"solar_start_current"`
		SolarStopTime     int `json:"solar_stop_time"`
	} `json:"settings"`
	EvMeter struct {
		ImportActiveEnergy float64 `json:"import_active_energy"`
		ImportActivePower  float64 `json:"import_active_power"`
	} `json:"ev_meter"`
	MainsMeter struct {
		ImportActiveEnergy float64 `json:"import_active_energy"`
		ExportActiveEnergy float64 `json:"export_active_energy"`
	} `json:"mains_meter"`
	PhaseCurrents struct {
		Total float64 `json:"TOTAL"`
		L1    float64 `json:"L1"`
		L2    float64 `json:"L2"`
		L3    float64 `json:"L3"`
	} `json:"phase_currents"`
}

// SmartEVSE-3.5 operating mode IDs
const (
	smartEvse3ModeOff    = 0
	smartEvse3ModeNormal = 1
	smartEvse3ModeSolar  = 2
	smartEvse3ModeSmart  = 3
	smartEvse3ModePause  = 4
)

func init() {
	registry.Add("smart-evse", NewSmartEVSE3FromConfig)
}

// NewSmartEVSE3FromConfig creates a SmartEVSE-3.5 REST charger from generic config
func NewSmartEVSE3FromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		URI   string
		Cache time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, fmt.Errorf("missing uri")
	}

	return NewSmartEVSE3(cc.URI, cc.Cache)
}

// NewSmartEVSE3 creates a new SmartEVSE-3.5 REST charger
func NewSmartEVSE3(uri string, cache time.Duration) (*SmartEVSE3, error) {
	log := util.NewLogger("smart-evse")

	wb := &SmartEVSE3{
		Helper: request.NewHelper(log),
		uri:    strings.TrimRight(util.DefaultScheme(uri, "http"), "/"),
		curr:   60, // 6 A in 1/10 A
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
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

	// force NORMAL mode so override_current takes effect
	if res.ModeID != smartEvse3ModeNormal {
		if err := wb.setMode(smartEvse3ModeNormal); err != nil {
			return nil, err
		}
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
	case 0: // STATE_A
		return api.StatusA, nil
	case 1, 4, 5, 8, 9, 11, 12, 13, 14:
		// STATE_B, STATE_COMM_B, STATE_COMM_B_OK, STATE_ACTSTART, STATE_B1,
		// STATE_MODEM_REQUEST, STATE_MODEM_WAIT, STATE_MODEM_DONE, STATE_MODEM_DENIED
		return api.StatusB, nil
	case 2, 3, 6, 7, 10:
		// STATE_C, STATE_D, STATE_COMM_C, STATE_COMM_C_OK, STATE_C1
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid state: %d (%s)", res.Evse.StateID, res.Evse.State)
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

	if enable && res.ModeID != smartEvse3ModeNormal {
		return wb.setMode(smartEvse3ModeNormal)
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

var _ api.Meter = (*SmartEVSE3)(nil)

// CurrentPower implements the api.Meter interface
func (wb *SmartEVSE3) CurrentPower() (float64, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return 0, err
	}

	// import_active_power is reported in W when an EV meter is configured
	return res.EvMeter.ImportActivePower, nil
}

var _ api.MeterEnergy = (*SmartEVSE3)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *SmartEVSE3) TotalEnergy() (float64, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return 0, err
	}

	// import_active_energy is reported in Wh
	return res.EvMeter.ImportActiveEnergy / 1e3, nil
}

var _ api.PhaseCurrents = (*SmartEVSE3)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *SmartEVSE3) Currents() (float64, float64, float64, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	// phase currents are reported in 1/10 A
	return res.PhaseCurrents.L1 / 10, res.PhaseCurrents.L2 / 10, res.PhaseCurrents.L3 / 10, nil
}

var _ api.Identifier = (*SmartEVSE3)(nil)

// Identify implements the api.Identifier interface
func (wb *SmartEVSE3) Identify() (string, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return "", err
	}

	return res.Evse.RFID, nil
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
	fmt.Printf("\tEVSE state: %s (%d)\n", res.Evse.State, res.Evse.StateID)
	fmt.Printf("\tEVSE error: %s (%d)\n", res.Evse.Error, res.Evse.ErrorID)
	fmt.Printf("\tCharge current: %d A\n", res.Settings.ChargeCurrent)
	fmt.Printf("\tOverride current: %.1f A\n", float64(res.Settings.OverrideCurrent)/10)
	fmt.Printf("\tCurrent min/max: %d/%d A\n", res.Settings.CurrentMin, res.Settings.CurrentMax)
	fmt.Printf("\tTemperature: %d °C\n", res.Evse.Temp)
}
