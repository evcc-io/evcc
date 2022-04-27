package charger

// LICENSE

// Copyright (c) 2019-2022 andig

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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/smaevcharger"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

//Constants
const (
	smaevchNConNCarNChar = float32(200111) // No Car connectec and no charging
	smaevchYConYCarNChar = float32(200112) // Car connected and no charging
	smaevchYConYCarYChar = float32(200113) // Car connected and charging

	smaevchFastCharge = "4718" // Schnellladen - 4718
	smaevchOptiCharge = "4719" // Optimiertes Laden - 4719
	smaevchPlanCharge = "4720" // Laden mit Vorgabe - 4720
	smaevchStopCharge = "4721" // Ladestopp - 4721

	smaevchSwitchOeko = float32(4950) // Switch in PV Loading (Can be Optimized or Planned PV loading)
	smaevchSwitchFast = float32(4718) // Switch in Fast Charge Mode
)

// smaevchager charger implementation
type Smaevcharger struct {
	*request.Helper
	host      	string	// 192.168.XXX.XXX
	user     	string	// LOGIN user
	password 	string	// password
	Auth				smaevcharger.AuthToken
	MeasurementsData	[]smaevcharger.Measurements
	ParametersData		[]smaevcharger.Parameters	
}

func init() {
	registry.Add("smaevcharger", NewSmaevchargerFromConfig)
}

// NewSmaevchargerFromConfig creates a Smaevcharger charger from generic config
func NewSmaevchargerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Host     string
		User     string
		Password string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Host == "" {
		return nil, errors.New("missing host")
	}
	if cc.User == "" {
		return nil, errors.New("missing user")
	} else if cc.User == "admin" {
		return nil, errors.New("user admin not allowed, create new user")
	}
	if cc.Password == "" {
		return nil, errors.New("missing password")
	}
	
	return NewSmaevcharger(cc.Host, cc.User, cc.Password)
}

// NewSmaevcharger creates Smaevcharger charger
func NewSmaevcharger(host string, user string, password string) (api.Charger, error) {
	wb := &Smaevcharger{
		Helper:		request.NewHelper(util.NewLogger("smaevcharger")),
		host:      	"http://" + host + "/api/v1",
		user:     	user,
		password: 	password,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Smaevcharger) Status() (api.ChargeStatus, error) {
	StateChargerCharging := wb.GetMeasurement("Measurement.Operation.EVeh.ChaStt")

	switch StateChargerCharging {
	case smaevchNConNCarNChar:// No Car connectec and no charging
		return api.StatusA, nil
	case smaevchYConYCarNChar:// Car connected and no charging
		return api.StatusB, nil
	case smaevchYConYCarYChar:// Car connected and charging
		return api.StatusC, nil
	}
		return api.StatusNone, fmt.Errorf("unkown charger state: %s", StateChargerCharging)
}

// Enabled implements the api.Charger interface
func (wb *Smaevcharger) Enabled() (bool, error) {
	StateChargerMode := wb.GetParameter("Parameter.Chrg.ActChaMod")

	switch StateChargerMode {
	case smaevchFastCharge:// Schnellladen - 4718
		return true, nil
	case smaevchOptiCharge:// Optimiertes Laden - 4719
		return true, nil
	case smaevchPlanCharge:// Laden mit Vorgabe - 4720
		return true, nil
	case smaevchStopCharge:// Ladestopp - 4721
		return false, nil
	}
	return false, fmt.Errorf("unknown charger charge mode: %s", StateChargerMode)
}

// Enable implements the api.Charger interface
func (wb *Smaevcharger) Enable(enable bool) error {
	StateChargerSwitch := wb.GetMeasurement("Measurement.Chrg.ModSw")
	if enable {
		switch StateChargerSwitch {
		case smaevchSwitchOeko:// Switch PV Loading
			wb.SendParameter("Parameter.Chrg.ActChaMod", smaevchOptiCharge)
		case smaevchSwitchFast:// Fast charging
			wb.SendParameter("Parameter.Chrg.ActChaMod", smaevchFastCharge)
		}
	} else {
		wb.SendParameter("Parameter.Chrg.ActChaMod", smaevchStopCharge)
	}
	time.Sleep(time.Second) //Some Delay to prevent out of Sync - The Charger needs some time to react after setting have been changed
	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *Smaevcharger) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %v", current)
	}
	wb.SendParameter("Parameter.PCC.ARtg", strconv.FormatInt(int64(current), 10))
	//This Parameter should be enough to change the Chargerates, but SMA Software seems to be very buggy and ignores this value
	//aditionally it is still possible to set the current indirect via the max watts which will internally change the charger behavior
	//Convert current to watts
	ChargeRate := current * 3 * 230
	//Check limits - ChargeRates above 11kW are illegal in Germany with funding (can be an option in the future)
	if ChargeRate > 11000 {
		ChargeRate = 11000
	}
	// Set both Max in and Max out watts, Charger seems to be a bit buggy if only one is changed
	wb.SendParameter("Parameter.Inverter.WMax", strconv.FormatInt(int64(ChargeRate), 10))
	wb.SendParameter("Parameter.Inverter.WMaxIn", strconv.FormatInt(int64(ChargeRate), 10))
	return nil
}

var _ api.ChargerEx = (*Smaevcharger)(nil)


// maxCurrentMillis implements the api.ChargerEx interface
func (wb *Smaevcharger) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}
	wb.SendParameter("Parameter.PCC.ARtg", strconv.FormatFloat(current,'f', 2, 64))
	//This Parameter should be enough to change the Chargerates, but SMA Software seems to be very buggy and ignores this value
	//aditionally it is still possible to set the current indirect via the max watts which will internally change the charger behavior
	//Convert current to watts
	ChargeRate := current * 3 * 230
	//Check limits - ChargeRates above 11kW are illegal in Germany with funding (can be an option in the future)
	if ChargeRate > 11000 {
		ChargeRate = 11000
	}
	// Set both Max in and Max out watts, Charger seems to be a bit buggy if only one is changed
	wb.SendParameter("Parameter.Inverter.WMax", strconv.FormatInt(int64(ChargeRate), 10))
	wb.SendParameter("Parameter.Inverter.WMaxIn", strconv.FormatInt(int64(ChargeRate), 10))
	return nil
}

var _ api.Meter = (*Smaevcharger)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Smaevcharger) CurrentPower() (float64, error) {
	var Power = wb.ConvertInterfaceToFloat(wb.GetMeasurement("Measurement.Metering.GridMs.TotWIn"))
	return Power, nil
}

var _ api.ChargeRater = (*Smaevcharger)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Smaevcharger) ChargedEnergy() (float64, error) {
	var data = wb.ConvertInterfaceToFloat(wb.GetMeasurement("Measurement.ChaSess.WhIn"))
	return float64(data / 1000.0), nil
}

var _ api.MeterCurrent = (*Smaevcharger)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Smaevcharger) Currents() (float64, float64, float64, error) {
	var PhsA = wb.ConvertInterfaceToFloat(wb.GetMeasurement("Measurement.GridMs.A.phsA"))
	var PhsB = wb.ConvertInterfaceToFloat(wb.GetMeasurement("Measurement.GridMs.A.phsB"))
	var PhsC = wb.ConvertInterfaceToFloat(wb.GetMeasurement("Measurement.GridMs.A.phsC"))
	return PhsA, PhsB, PhsC, nil
}

func (wb *Smaevcharger) GetAuthToken() bool {
	client := &http.Client{}
	URL := wb.host + "/token"
	v := url.Values{}
	v.Set("grant_type", "password")
	v.Set("password", wb.password)
	v.Set("username", wb.user)
	//pass the values to the request's body
	req, err := http.NewRequest("POST", URL, strings.NewReader(v.Encode()))
	if err != nil {
		return false
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	reader := strings.NewReader(string(respBody))
	err = json.NewDecoder(reader).Decode(&wb.Auth)
	
	return err == nil
}

func (wb *Smaevcharger)GetMeasurementData() bool {
	
	tokenok := wb.GetAuthToken()

	if !tokenok{
		return false
	}
	client := &http.Client{}
	URL := wb.host + "/measurements/live"
	jsonData := []byte(`[{"componentId": "IGULD:SELF"}]`)
	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return false
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+ wb.Auth.Access_token )
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return false
	}
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		err = json.Unmarshal([]byte(respBody), &wb.MeasurementsData)
		return err == nil
	}
	return false
}

func (wb *Smaevcharger)GetParameterData() bool {

	tokenok := wb.GetAuthToken()

	if !tokenok{
		return false
	}

	client := &http.Client{}
	URL := wb.host + "/parameters/search/"
	jsonData := []byte(`{"queryItems":[{"componentId":"IGULD:SELF"}]}`)
	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return false
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+wb.Auth.Access_token)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return false
	}
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		err = json.Unmarshal([]byte(respBody), &wb.ParametersData)
		return err == nil
	}
	return false
}

func (wb *Smaevcharger)GetMeasurement(id string) interface{} {
	dataok := wb.GetMeasurementData()
	if !dataok {
		nil := fmt.Errorf("failed to aquire measurement data")
		return nil
	}
	var returndata interface{}

	for i := range wb.MeasurementsData {
		if wb.MeasurementsData[i].ChannelId == id {
			returndata = wb.MeasurementsData[i].Values[0].Value
			return returndata
		}
	}
	return returndata
}

func (wb *Smaevcharger)GetParameter(id string) interface{} {
	
	dataok := wb.GetParameterData()
	if !dataok {
		nil := fmt.Errorf("failed to aquire parameter data")
		return nil
	}
	var returndata interface{}

	for i := range wb.ParametersData[0].Values {
		if wb.ParametersData[0].Values[i].ChannelId == id {
			returndata = wb.ParametersData[0].Values[i].Value
			return returndata
		}
	}
	return returndata
}

func (wb *Smaevcharger)SendParameter(id string, value string) bool {
	wb.GetParameterData()
	var parameter smaevcharger.SendParameter
	var data smaevcharger.SendData

	data.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	data.ChannelId = id
	data.Value = value
	parameter.Values = append(parameter.Values, data)

	parameterAvaliable := false
	for i := range wb.ParametersData[0].Values {
		if wb.ParametersData[0].Values[i].ChannelId == id {
			parameterAvaliable = true
			break
		}
	}

	if parameterAvaliable {

		client := &http.Client{}
		URL := wb.host + "/parameters/IGULD:SELF/"
		jsonData, err := json.Marshal(parameter)
		if err != nil {
			return false
		}	
		req, err := http.NewRequest("PUT", URL, bytes.NewBuffer(jsonData))
		if err != nil {
			return false
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Bearer "+wb.Auth.Access_token)
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			return true
		} else {
			return false
		}
	}
	return false
}

func (wb *Smaevcharger)ConvertInterfaceToFloat(data interface{})float64{
	var dataout float64
	switch value := data.(type) {
	case float32:
		dataout = float64(value)
	}
	return dataout
}